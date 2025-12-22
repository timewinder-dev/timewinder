# Bounded Buffer (Producer-Consumer) in Timewinder
# Converted from PlusCal example
# Source: https://github.com/belaban/pluscal/blob/master/BoundedBuffer.tla
# Original: Models producers and consumers sharing a bounded buffer

BUF_SIZE = 3
MAX_ITEMS = 10

buf = []
produced = 0
consumed = 0

def producer():
    while produced < MAX_ITEMS:
        # Try to produce
        step("try_produce")

        if len(buf) < BUF_SIZE:
            produced = produced + 1
            buf.append(produced)

def consumer():
    while consumed < MAX_ITEMS:
        # Try to consume
        step("try_consume")

        if len(buf) > 0:
            buf.pop(0)
            consumed = consumed + 1
