# Simple Chat - Discord Alternative

A minimal real-time chat application built with Go and TailwindCSS that you can run locally and connect with others across different networks.

## Features

- 💬 Real-time messaging using WebSockets
- 👥 Multiple users support
- 🌐 Cross-network connectivity
- 🎨 Clean, Discord-like UI with TailwindCSS
- 🔄 Auto-reconnection
- 📱 Responsive design

## Prerequisites

- Go 1.21 or higher
- Internet connection (for initial TailwindCSS CDN)

## Installation

1. Navigate to the project directory:
```bash
cd simple-chat
```

2. Install dependencies:
```bash
go mod download
```

## Running Locally

1. Start the server:
```bash
go run main.go
```

2. Open your browser and navigate to:
```
http://localhost:8080
```

3. Enter a username and start chatting!

## Connecting Across Networks

To allow others on different networks to connect to your chat server:

### Option 1: Port Forwarding (Recommended for home networks)

1. Find your local IP address:
   - Windows: `ipconfig`
   - Mac/Linux: `ifconfig` or `ip addr`

2. Set up port forwarding on your router:
   - Forward port 8080 to your computer's local IP
   - Each router interface is different, consult your router's manual

3. Find your public IP:
   - Visit: https://whatismyipaddress.com/

4. Share your public IP with others:
   - They can connect at: `http://YOUR_PUBLIC_IP:8080`

### Option 2: Using ngrok (Easiest for testing)

1. Install ngrok: https://ngrok.com/download

2. Run ngrok:
```bash
ngrok http 8080
```

3. Share the ngrok URL (e.g., `https://abc123.ngrok.io`) with others

### Option 3: Deploy to a VPS

Deploy to a cloud server (DigitalOcean, AWS, etc.) for permanent availability:

```bash
# On your server
git clone github.com/NathanielRand/notcord
cd notcord
go build
./notcord
```

Configure your firewall to allow port 8080.

## Configuration

To change the port, edit `main.go`:

```go
log.Fatal(http.ListenAndServe(":8080", nil))
// Change :8080 to your desired port
```

## Security Notes

⚠️ **Important**: This is an MVP and lacks authentication, encryption, and other security features. For production use, consider:

- Adding TLS/HTTPS
- Implementing user authentication
- Adding rate limiting
- Sanitizing user input
- Adding message persistence
- Implementing private channels/rooms

## Architecture

- **Backend**: Go with Gorilla WebSocket
- **Frontend**: HTML + TailwindCSS + Vanilla JavaScript
- **Real-time**: WebSocket connections for bi-directional communication
- **Concurrency**: Go routines for handling multiple clients

## File Structure

```
.
├── main.go           # Go server with WebSocket handling
├── go.mod            # Go module dependencies
├── static/
│   └── index.html    # Frontend UI
└── README.md         # This file
```

## Troubleshooting

**Can't connect from other networks:**
- Check firewall settings
- Verify port forwarding is configured correctly
- Ensure the server is running and accessible locally first

**Messages not appearing:**
- Check browser console for errors
- Verify WebSocket connection is established
- Try refreshing the page

**Connection keeps dropping:**
- Check network stability
- Look for firewall interference
- Verify the server is still running

## Future Enhancements

- [ ] User authentication
- [ ] Multiple chat rooms/channels
- [ ] Message persistence (database)
- [ ] File/image sharing
- [ ] User presence indicators
- [ ] Message reactions
- [ ] Direct messages
- [ ] User avatars

## License

MIT License - Feel free to use and modify!
