"""Receipt rendering for an order that has already been priced."""

from typing import List

from app.models import LineItem, Order
from app.service import TAX_RATE, compute_total


def line_total(item: LineItem) -> float:
    """The charged amount for one line, tax included."""
    return compute_total([item], TAX_RATE, "USD")


def line_text(item: LineItem) -> str:
    """One printable receipt line."""
    return "{0} x{1}  {2:.2f}".format(item.sku, item.quantity, line_total(item))


def render_lines(order: Order) -> List[str]:
    """Every printable line of a receipt, in order."""
    return [line_text(item) for item in order.items]
