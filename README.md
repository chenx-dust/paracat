# ParaCat: A multipath UDP forwarder for high reliability/throughput

In crowded Internet, all connections are not reliable. To minimize jitter and packet loss, we can send it through different routes simultaneously then get redundancy.

## Structure

```mermaid
flowchart LR
    C[UDP Client]
    IN[Inbound]
    OUT[Outbound]
    R0(Relay Server #0)
    R1(Relay Server #1)
    Rn(Relay Server #n)
    S(UDP Server)

    C -->|UDP| IN

    IN -->|Raw TCP| R0
    IN -->|Raw UDP| R0
    IN -->|SOCKS5| R1
    IN -->|Others| Rn
    IN -->|Raw TCP| OUT
    IN -->|Raw UDP| OUT

    R0 --> OUT
    R1 --> OUT
    Rn --> OUT

    OUT -->|UDP| S
```

## TODO

- [X] Round-robin mode
- [ ] Remove unused UDP connections
- [ ] Optimize delay
- [ ] Re-connect after EOF
- [ ] Congestion control algorithm
- [ ] Fake TCP with eBPF
- [ ] GRO & GSO
- [ ] UDP MTU Discovery
