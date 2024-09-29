
# Implementation Notes

This document explains the choices made when I was building the SageSlayerServer.

## Proof-of-Work (PoW) Choice

### Why ECC-based PoW?

I chose Elliptic Curve Cryptography (ECC) for the PoW system because:

1. **Efficiency**: ECC provides strong security with smaller key sizes compared to algorithms like RSA. This means clients can complete the PoW faster but still need to use enough computational power to protect the server.
2. **DDoS Protection**: By forcing clients to spend computing power before they can interact with the server, ECC-based PoW helps prevent denial-of-service (DoS) attacks.
3. **Scalability**: The difficulty level can be adjusted dynamically based on the client's behavior, like increasing difficulty for clients who send too many requests or make errors.

### Comparison with Hashcash

- **Hashcash**: Hashcash is simpler but uses brute-force hashing (like Bitcoin mining). While effective, it's slower and less flexible compared to ECC. The computational power required grows significantly with difficulty, making it harder to adjust in real-time without causing major slowdowns.
- **ECC PoW**: ECC provides a smoother adjustment of difficulty levels and is more efficient in terms of security per operation, meaning clients use less power for the same level of security.
- **ASIC resistance**: ECC-based PoW offer better ASIC resistance.

## Protocol Buffers (Protobuf) Serialization

### Why Protobuf?

1. **Speed and Size**: Protobuf creates very small and fast binary messages, which is important when many clients are connecting at once.
2. **Cross-platform**: It works across multiple programming languages, which is useful if we ever want to write clients in languages other than Go.
3. **Easy to Update**: Protobuf is great for versioning. If we need to add new fields or change how messages are structured, Protobuf makes sure older versions can still work without breaking.

## Blacklist Mechanism

I added a blacklist to handle clients who are behaving badly (like sending too many invalid messages or failing PoW). Why:

1. **Stops Malicious Clients**: It blocks clients who are sending too many errors or spammy requests. This prevents them from overloading the server.
2. **Temporary Ban**: Instead of permanently banning clients, the blacklist blocks them for a certain time, allowing them to try again later.

## Rate Limiter

The rate limiter controls how fast clients can send requests. If clients send too many requests in a short period, they get limited. Here's why:

1. **Prevents DoS Attacks**: It makes sure no single client can overwhelm the server with too many requests.
2. **Fairness**: All clients get equal access to server resources, stopping any one client from using up too much.
3. **Decay Over Time**: Clients request and error rates gradually go down if they stop sending too many requests, which means they aren't permanently locked out for mistakes.