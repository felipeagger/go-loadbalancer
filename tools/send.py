import socket
import sys

if len(sys.argv) < 3:
    print("Usage: echo 'data' | python3 send.py <HOST> <PORT>")
    sys.exit(1)

HOST = sys.argv[1]
PORT = int(sys.argv[2])

try:
    data_to_send = sys.stdin.read().strip()
    if not data_to_send:
        print("Error: No data received from standard input.")
        sys.exit(1)
except Exception as e:
    print(f"Error reading from standard input: {e}")
    sys.exit(1)

data_to_send_bytes = data_to_send.encode('utf-8')

try:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.connect((HOST, PORT))
        
        s.sendall(data_to_send_bytes)
        response = s.recv(1024)
        
        print(f"Response: '{response.decode().strip()}'")

except ConnectionRefusedError:
    print(f"Error: Connection refused. The server is not running on {HOST}:{PORT}.")
except Exception as e:
    print(f"Error: {e}")
