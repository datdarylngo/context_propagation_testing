document.addEventListener('DOMContentLoaded', function() {
    const userInput = document.getElementById('user-input');
    const sendButton = document.getElementById('send-button');
    const chatHistory = document.getElementById('chat-history');
    const logOutput = document.getElementById('log-output');
    const eventSource = new EventSource('/logs');

    function addMessage(text, isUser = false) {
        const messageDiv = document.createElement('div');
        messageDiv.className = `message ${isUser ? 'user' : 'system'}`;
        messageDiv.textContent = text;
        chatHistory.appendChild(messageDiv);
        chatHistory.scrollTop = chatHistory.scrollHeight;
    }

    function addLogEntry(data) {
        const entry = document.createElement('div');
        
        if (data.type === 'span') {
            entry.className = 'log-entry span';
            entry.textContent = `Span: ${data.name} (Trace: ${data.trace_id})`;
            
            // Add attributes
            for (const [key, value] of Object.entries(data.attributes)) {
                const attrDiv = document.createElement('div');
                attrDiv.className = 'log-entry attribute';
                attrDiv.textContent = `${key}: ${value}`;
                entry.appendChild(attrDiv);
            }
        }
        
        logOutput.appendChild(entry);
        logOutput.scrollTop = logOutput.scrollHeight;
    }

    eventSource.onmessage = function(e) {
        const data = JSON.parse(e.data);
        if (data.type !== 'heartbeat') {
            addLogEntry(data);
        }
    };

    async function sendMessage() {
        const text = userInput.value.trim();
        if (!text) return;

        // Add user message to chat
        addMessage(text, true);
        userInput.value = '';

        try {
            const response = await fetch('/process', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ text: text })
            });

            const data = await response.json();
            addMessage(data.response);
        } catch (error) {
            console.error('Error:', error);
            addMessage('Error processing request');
        }
    }

    document.getElementById('send-to-llm-button').addEventListener('click', async function() {
        const text = document.getElementById('user-input').value.trim();
        if (!text) return;

        addMessage(text, true);
        document.getElementById('user-input').value = '';

        try {
            const response = await fetch('/send-to-llm', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ text: text })
            });

            const data = await response.json();
            if (data.error) {
                addMessage(`Error: ${data.error}`);
            } else {
                addMessage(data.response);
            }
        } catch (error) {
            console.error('Error:', error);
            addMessage('Error sending to LLM Gateway');
        }
    });

    sendButton.addEventListener('click', sendMessage);
    userInput.addEventListener('keypress', function(e) {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendMessage();
        }
    });
}); 