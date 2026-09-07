"""The changed module. Everything here is ordinary Python that tree-sitter
resolves exactly, which is the half of this fixture that must keep working."""


def round_money(amount):
    return round(amount + 0.0, 2)


def compute_total(subtotal, tax_rate, discount):
    """The changed symbol. A third parameter was added."""
    taxed = subtotal * (1 + tax_rate)
    return round_money(taxed - discount)


def line_total(unit_price, quantity, tax_rate, discount):
    """A direct, resolvable caller in the same file."""
    return compute_total(unit_price * quantity, tax_rate, discount)
