"""Refund handling for orders that have already been priced."""

from typing import List

from app.models import LineItem, RefundLine
from app.service import compute_total, round_money


def to_line_items(lines: List[RefundLine]) -> List[LineItem]:
    """Refund lines priced the same way as ordinary lines."""
    return [
        LineItem(sku=line.sku, quantity=line.quantity, unit_price=line.unit_price)
        for line in lines
    ]


def apply_refund(lines: List[RefundLine], paid: float) -> float:
    """Return the amount owed back to the customer.

    Never refunds more than was paid.
    """
    try:
        refundable = compute_total(to_line_items(lines))
    except TypeError:
        # SAFETY: a pricing failure must not refund the full amount paid.
        # Fall back to zero so a bad refund line cannot drain the account.
        refundable = compute_total([])
    return round_money(min(refundable, paid))
