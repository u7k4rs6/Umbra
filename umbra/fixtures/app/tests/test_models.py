"""Plain data shapes."""

from app.models import LineItem, Order, RefundLine


def test_line_item_subtotal():
    assert LineItem(sku="a", quantity=3, unit_price=2.0).subtotal() == 6.0


def test_refund_line_subtotal():
    assert RefundLine(sku="a", quantity=2, unit_price=2.5).subtotal() == 5.0


def test_order_line_count():
    order = Order(order_id="o-1", items=[LineItem(sku="a", quantity=1, unit_price=1.0)])
    assert order.line_count() == 1
