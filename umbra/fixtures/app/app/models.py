"""Data shapes for the order service."""

from dataclasses import dataclass, field
from typing import List


@dataclass
class LineItem:
    """One purchasable line on an order."""

    sku: str
    quantity: int
    unit_price: float

    def subtotal(self) -> float:
        return self.quantity * self.unit_price


@dataclass
class RefundLine:
    """One line being refunded, pointing back at the original item."""

    sku: str
    quantity: int
    unit_price: float
    reason: str = ""

    def subtotal(self) -> float:
        return self.quantity * self.unit_price


@dataclass
class Order:
    """An order and the lines it carries."""

    order_id: str
    items: List[LineItem] = field(default_factory=list)
    currency: str = "USD"

    def line_count(self) -> int:
        return len(self.items)
