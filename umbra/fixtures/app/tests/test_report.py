"""Receipt rendering. This file reaches compute_total four hops away."""

from app.models import LineItem, Order
from app.report import render_lines


def test_receipt_total():
    order = Order(order_id="o-1", items=[LineItem(sku="a", quantity=2, unit_price=5.0)])
    assert render_lines(order) == ["a x2  10.80"]


def test_receipt_is_empty_without_items():
    assert render_lines(Order(order_id="o-2")) == []
