"""Billing. The session never opens this file, so everything in it is in
shadow, and the graph resolves only part of it.

RateCard.apply calls the changed symbol through an import, which the provider
resolves. quote calls that method on a receiver it has to infer. invoice calls
quote by name, so its own hop is exact and the chain it hangs off is not.
"""

from pricing import compute_total


class RateCard:
    """A rate card whose method is reached through an inferred receiver."""

    def __init__(self, tax_rate=0.1):
        self.tax_rate = tax_rate

    def apply(self, subtotal, discount):
        """A direct, resolvable caller of the changed symbol."""
        return compute_total(subtotal, self.tax_rate, discount)


def quote(subtotal, discount):
    """Calls the method on a freshly constructed receiver. The provider
    resolves this by inferring the constructor's type, not by parsing a
    definition, and reports it as type_inferred."""
    return RateCard().apply(subtotal, discount)


def invoice(subtotal, discount):
    """An ordinary, exactly resolved call to quote. It reaches the changed
    symbol only through the inferred hop above, which is what makes it a claim
    a reader has to check rather than one the graph can vouch for."""
    return quote(subtotal, discount)
