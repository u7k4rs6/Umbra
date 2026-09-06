"""Refunds, including the capped path."""

from app.models import RefundLine
from app.refunds import apply_refund, to_line_items


def test_to_line_items_preserves_price():
    lines = [RefundLine(sku="a", quantity=1, unit_price=4.0)]
    items = to_line_items(lines)
    assert items[0].unit_price == 4.0


def test_apply_refund_caps_at_paid():
    lines = [RefundLine(sku="a", quantity=10, unit_price=100.0)]
    assert apply_refund(lines, paid=50.0) == 50.0


def test_apply_refund_normal():
    lines = [RefundLine(sku="a", quantity=1, unit_price=10.0)]
    assert apply_refund(lines, paid=999.0) == 10.80
