document.addEventListener('DOMContentLoaded', function() {
    // Get elements once at startup
    const userInput = document.getElementById('user-input');
    const sendButton = document.getElementById('send-button');
    const chatHistory = document.getElementById('chat-history');
    const logOutput = document.getElementById('log-output');

    // Verify elements exist
    if (!userInput || !sendButton || !chatHistory || !logOutput) {
        console.error('Required elements not found');
        return;
    }

    // Simple message display
    function addMessage(text, isUser) {
        const div = document.createElement('div');
        div.innerHTML = text.replace(/\n/g, '<br>');
        div.className = isUser ? 'message user' : 'message system';
        chatHistory.appendChild(div);
        chatHistory.scrollTop = chatHistory.scrollHeight;
    }

    // Add log entry
    function addLogEntry(text, type = 'info') {
        const entry = document.createElement('div');
        entry.className = `log-entry ${type}`;
        
        // Try to parse as JSON for span data
        try {
            const data = JSON.parse(text);
            if (data.type === 'span') {
                entry.innerHTML = `
                    <div class="span-header">Span: ${data.name}</div>
                    <div class="span-id">TraceID: ${data.trace_id}</div>
                    <div class="span-id">SpanID: ${data.span_id}</div>
                    <div class="span-time">Start: ${data.start}</div>
                    <div class="span-time">End: ${data.end}</div>
                `;

                // Format attributes properly
                if (data.attributes) {
                    const attrs = data.attributes;
                    Object.entries(attrs).forEach(([key, value]) => {
                        const attrDiv = document.createElement('div');
                        attrDiv.className = 'log-entry attribute';
                        // Handle attribute value object
                        if (typeof value === 'object' && value !== null) {
                            attrDiv.textContent = `${key}: ${value.Value}`;
                        } else {
                            attrDiv.textContent = `${key}: ${value}`;
                        }
                        entry.appendChild(attrDiv);
                    });
                }
            } else {
                entry.textContent = text;
            }
        } catch (e) {
            // Not JSON, treat as regular text
            entry.textContent = text;
        }
        
        logOutput.appendChild(entry);
        logOutput.scrollTop = logOutput.scrollHeight;
    }

    // Send message to server
    async function sendMessage() {
        const text = userInput.value.trim();
        if (!text) return;

        // Show user message
        addMessage(text, true);
        userInput.value = '';

        // Send to server
        try {
            const response = await fetch('/chat', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ message: text })
            });

            if (!response.ok) throw new Error('Server error');

            const data = await response.json();
            addMessage(data.response, false);
        } catch (error) {
            const errorMsg = 'Error: ' + error.message;
            addMessage(errorMsg, false);
            addLogEntry(errorMsg, 'error');
        }
    }

    // Add SSE listener for gRPC messages and spans
    const eventSource = new EventSource('/events');
    eventSource.onmessage = function(e) {
        console.log('SSE message received:', e.data);
        
        try {
            // Try to parse as JSON (span data)
            const data = JSON.parse(e.data);
            if (data.type === 'span') {
                addLogEntry(e.data, 'span');
            } else if (data.includes && data.includes('[From Python Service]')) {
                // Python service messages go to chat
                addMessage(data, false);
            }
        } catch (e) {
            // Not JSON, treat as regular message
            if (e.data.includes('[From Python Service]')) {
                addMessage(e.data, false);
            } else {
                addLogEntry(e.data, 'info');
            }
        }
    };

    // Event listeners
    sendButton.onclick = sendMessage;
    userInput.onkeypress = function(e) {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendMessage();
        }
    };

    // Initial log
    addLogEntry('LLM Gateway UI initialized', 'info');
}); 