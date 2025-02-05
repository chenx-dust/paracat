# Benchmark

## Environment

- CPU: AMD Ryzen 9 5900HX (8C in PVE)
- Memory: 8GB
- Network: Loopback

## Original

- 768.62 Mbps Bidirectional Transfer

## With Post-Filter Channel

- 1056.92 Mbps Bidirectional Transfer
- 1839.68 Mbps Forward Transfer
- 1784.70 Mbps Reverse Transfer

## With Double Channel Filter

- 1016.88 Mbps Bidirectional Transfer
- 1709.00 Mbps Forward Transfer
- 1662.63 Mbps Reverse Transfer

## With Single GRO

- 3731.67 Mbps / 280.14 Mbps Bidirectional Transfer
- 9129.70 Mbps Forward Transfer
- 460.15 Mbps Reverse Transfer

## With Full GRO

- 3634.26 Mbps Bidirectional Transfer
- 9320.89 Mbps Forward Transfer
- 9308.98 Mbps Reverse Transfer

## With GRO & GSO (before Optimized)

- 1968.95 Mbps Bidirectional Transfer
- 5227.55 Mbps Forward Transfer
- 5544.14 Mbps Reverse Transfer
