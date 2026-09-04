"""The HTTP-facing layer of the order service."""

from typing import Dict

from app.models import LineItem, Order
from app.service import TAX_RATE, compute_total, summarize


def health() -> Dict[str, str]:
    """Liveness response."""
    return {"status": "ok"}


def handle_order(order: Order) -> Dict[str, object]:
    """Price an order and return the payload the client receives."""
    total = compute_total(order.items, TAX_RATE, order.currency)
    return {
        "order_id": order.order_id,
        "currency": order.currency,
        "lines": order.line_count(),
        "total": total,
        "summary": summarize(order.items),
    }


def quote(sku: str, quantity: int, unit_price: float) -> float:
    """Price a single hypothetical line without creating an order."""
    item = LineItem(sku=sku, quantity=quantity, unit_price=unit_price)
    return compute_total([item], TAX_RATE, "USD")
