"""The API layer. This is the file the probe session runs."""

from app.api import handle_order, health, quote
from app.models import LineItem, Order


def test_health():
    assert health() == {"status": "ok"}


def test_handle_order():
    order = Order(order_id="o-1", items=[LineItem(sku="a", quantity=2, unit_price=5.0)])
    payload = handle_order(order)
    assert payload["order_id"] == "o-1"
    assert payload["lines"] == 1
    assert payload["total"] == 10.80


def test_quote_single_line():
    assert quote("a", 1, 10.0) == 10.80
