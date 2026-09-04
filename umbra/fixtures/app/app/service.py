"""Order totals and the rounding rules that go with them."""

from typing import Iterable, List

from app.models import LineItem

TAX_RATE = 0.08


def round_money(value: float) -> float:
    """Round to two places the way the ledger expects."""
    return round(value + 1e-9, 2)


def apply_tax(amount: float) -> float:
    """Add sales tax to an untaxed amount."""
    return round_money(amount * (1.0 + TAX_RATE))


def compute_total(items: Iterable[LineItem]) -> float:
    """Total a collection of line items, tax included.

    This is the symbol the fixture changes. It is called from app/api.py,
    app/refunds.py and tests/test_service.py.
    """
    subtotal = 0.0
    for item in items:
        subtotal += item.subtotal()
    return apply_tax(subtotal)


def summarize(items: List[LineItem]) -> str:
    """A short human readable total line."""
    total = compute_total(items)
    return "{0} items, {1:.2f}".format(len(items), total)
