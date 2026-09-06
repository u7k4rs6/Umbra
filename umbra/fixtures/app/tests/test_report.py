"""Receipt rendering. This file reaches compute_total four hops away."""

from app.models import LineItem, Order
from app.report import format_footer, render_lines


def test_receipt_total():
    order = Order(order_id="o-1", items=[LineItem(sku="a", quantity=2, unit_price=5.0)])
    assert render_lines(order) == ["a x2  10.80"]


def test_receipt_is_empty_without_items():
    assert render_lines(Order(order_id="o-2")) == []


def test_footer_pads_the_amount():
    order = Order(order_id="A2", items=[LineItem(sku="w", quantity=1, unit_price=3.0)])
    assert format_footer(order, 12.5) == "TOTAL 1        12.50"
