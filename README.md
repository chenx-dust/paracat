# ParaCat: A multipath UDP forwarder for high reliability/throughput

In crowded Internet, all connections are not reliable. To minimize jitter and packet loss, we can send it through different routes simultaneously then get redundancy.

## Structure

```mermaid
flowchart LR
    C[UDP Client]
    PC[Paracat Client]
    D{Data}
    PS[Paracat Server]
    S(UDP Server)

    C -->|"handleForward()"| PC
    C <-->|MultiPort<->SinglePort| PC
    PC -->|"handleReverse()"| C

    PC -->|"
    SendUDPLoop()
    SendTCPLoop()
    Dispatcher
    "| D
    PC -->|MultiPort<->| D
    D -->|"
    handleUDPRelayRecv()
    ReceiveTCPLoop()
    Filter
    "| PC

    D -->|"
    handleUDPListener()
    ReceiveTCPLoop()
    Filter
    "| PS
    D -->|<->SinglePort| PS
    PS -->|"
    SendUDPLoop()
    SendTCPLoop()
    Dispatcher
    "| D

    PS -->|"handleForward()"| S
    PS <-->|MultiPort<->SinglePort| S
    S -->|"handleReverse()"| PS
```

## TODO

- [X] Round-robin mode
- [X] Remove unused UDP connections
- [X] GRO & GSO
- [ ] Optimize delay
- [ ] Re-connect after EOF
- [ ] Congestion control algorithm
- [ ] Fake TCP with eBPF
- [ ] UDP MTU discovery with DF
- [ ] API interface
