from flask import Flask, render_template, request, jsonify, Response
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import ConsoleSpanExporter, BatchSpanProcessor
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from opentelemetry.propagate import inject
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator
import grpc
from concurrent import futures
import llm_service_pb2
import llm_service_pb2_grpc
import queue
import json
import time

# Create a queue for logs
log_queue = queue.Queue()

# Custom exporter that sends to both console and UI
class UISpanExporter(ConsoleSpanExporter):
    def export(self, spans):
        for span in spans:
            # Format span data
            span_data = {
                'type': 'span',
                'name': span.name,
                'trace_id': format(span.context.trace_id, '032x'),
                'span_id': format(span.context.span_id, '016x'),
                'attributes': {k: str(v) for k, v in span.attributes.items()}
            }
            # Send to UI via queue
            log_queue.put(json.dumps(span_data))
        # Also export to console
        super().export(spans)

# Set up OpenTelemetry with UI exporter
tracer_provider = TracerProvider()
ui_exporter = UISpanExporter()
span_processor = BatchSpanProcessor(ui_exporter)
tracer_provider.add_span_processor(span_processor)
trace.set_tracer_provider(tracer_provider)

# Create a tracer
tracer = trace.get_tracer("python-text-service")

app = Flask(__name__)
FlaskInstrumentor().instrument_app(app)

# gRPC client
llm_channel = grpc.insecure_channel('localhost:50051')
llm_client = llm_service_pb2_grpc.LLMServiceStub(llm_channel)

@app.route('/logs')
def logs():
    def generate():
        while True:
            try:
                # Get log from queue with timeout
                log = log_queue.get(timeout=1)
                yield f"data: {log}\n\n"
            except queue.Empty:
                # Send heartbeat to keep connection alive
                yield f"data: {json.dumps({'type': 'heartbeat'})}\n\n"
            time.sleep(0.1)
    
    return Response(generate(), mimetype='text/event-stream')

@app.route('/')
def home():
    return render_template('index.html')

@app.route('/process', methods=['POST'])
def process_text():
    with tracer.start_as_current_span("process_text") as span:
        data = request.get_json()
        user_text = data.get('text', '')
        
        # Add service name and other attributes to the span
        span.set_attribute("service.name", "Python Text Service")
        span.set_attribute("input.text", user_text)
        
        # Process the text (for now, just echo back)
        response_text = f"You said: {user_text}"
        span.set_attribute("output.text", response_text)
        
        response = {
            'response': response_text
        }
        
        return jsonify(response)

@app.route('/send-to-llm', methods=['POST'])
def send_to_llm():
    with tracer.start_as_current_span("send_to_llm") as span:
        data = request.get_json()
        text = data.get('text', '')
        
        # Add service name and other attributes to the span
        span.set_attribute("service.name", "Python Text Service")
        span.set_attribute("input.text", text)
        
        try:
            # Create gRPC metadata with trace context
            metadata = []
            carrier = {}
            TraceContextTextMapPropagator().inject(carrier)
            for k, v in carrier.items():
                metadata.append((k.lower(), v))

            # Call Go service via gRPC with context
            response = llm_client.ProcessText(
                llm_service_pb2.TextRequest(text=text),
                metadata=metadata
            )
            span.set_attribute("output.text", response.response)
            return jsonify({'response': response.response})
        except Exception as e:
            span.record_exception(e)
            return jsonify({'error': str(e)}), 500

if __name__ == '__main__':
    app.run(port=5000, debug=True) 