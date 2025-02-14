# Benchmark

## Environment

- CPU: AMD Ryzen 9 5900HX (8C in PVE)
- Memory: 8GB
- Network: Loopback

## Original

- 768.62 Mbps Bidirectional Transfer

## With Post-Gather Channel

- 1056.92 Mbps Bidirectional Transfer
- 1839.68 Mbps Forward Transfer
- 1784.70 Mbps Reverse Transfer

## With Double Channel Gather

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

## With GRO & GSO (after Optimized)

- 4069.50 Mbps Bidirectional Transfer
- 13447.39 Mbps Forward Transfer
- 10025.05 Mbps Reverse Transfer

## With GRO & GSO (with smart pointer & buffer pool)

- 8874.25 Mbps Bidirectional Transfer
- 35520.84 Mbps Forward Transfer
- 41044.26 Mbps Reverse Transfer

## Original TCP

- 6979.40 Mbps Bidirectional Transfer
- 12442.19 Mbps Forward Transfer
- 13009.04 Mbps Reverse Transfer

## With large buffer reader TCP

- 677.77 Mbps Bidirectional Transfer (?)
- 15459.34 Mbps Forward Transfer
- 16360.45 Mbps Reverse Transfer

## Optimized TCP

- 1110.90 Mbps Bidirectional Transfer (high variance)
- 24471.42 Mbps Forward Transfer
- 23794.73 Mbps Reverse Transfer
