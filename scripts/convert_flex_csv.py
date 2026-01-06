#!/usr/bin/env python3
"""
Convert Flex CSV (Interactive Brokers) into Xero BankTransactions JSON.

Example:
  scripts/convert_flex_csv.py /path/to/flex.csv \
    --contact "Interactive Brokers" \
    --account-code 400 \
    --bank-account-code 090 \
    --pretty \
    | xero banktransactions create --file -
"""

from __future__ import annotations

import argparse
import csv
import json
import sys
from datetime import datetime
from decimal import Decimal, InvalidOperation


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Convert Flex CSV to Xero BankTransactions JSON"
    )
    parser.add_argument("csv_path", help="Path to Flex CSV file")
    parser.add_argument(
        "--contact",
        help="Contact name to use for all transactions",
    )
    parser.add_argument(
        "--contact-from",
        dest="contact_from",
        help="CSV column name to pull Contact name from",
    )
    parser.add_argument(
        "--account-code",
        required=True,
        help="Xero account code for line items",
    )
    parser.add_argument(
        "--bank-account-code",
        help="Xero bank account code (BankAccount.Code)",
    )
    parser.add_argument(
        "--date-column",
        default="SettleDate",
        help="CSV column name for transaction date (default: SettleDate)",
    )
    parser.add_argument(
        "--amount-column",
        default="Amount",
        help="CSV column name for amount (default: Amount)",
    )
    parser.add_argument(
        "--description-column",
        default="Description",
        help="CSV column name for description (default: Description)",
    )
    parser.add_argument(
        "--reference-column",
        default="TransactionID",
        help="CSV column name for reference (default: TransactionID)",
    )
    parser.add_argument(
        "--type-column",
        default="Type",
        help="CSV column name for type/label (default: Type)",
    )
    parser.add_argument(
        "--force-type",
        choices=["RECEIVE", "SPEND"],
        help="Override transaction type for all rows",
    )
    parser.add_argument(
        "--pretty",
        action="store_true",
        help="Pretty-print JSON",
    )
    parser.add_argument(
        "--strict",
        action="store_true",
        help="Exit non-zero if any row fails to parse",
    )
    return parser.parse_args()


def warn(msg: str) -> None:
    print(f"warning: {msg}", file=sys.stderr)


def parse_date(raw: str) -> str:
    if not raw:
        raise ValueError("empty date")
    value = raw.strip()
    if ";" in value:
        value = value.split(";", 1)[0]
    if "-" in value and len(value) == 10:
        # Assume YYYY-MM-DD
        return value
    if len(value) == 8 and value.isdigit():
        dt = datetime.strptime(value, "%Y%m%d")
        return dt.strftime("%Y-%m-%d")
    raise ValueError(f"unsupported date format: {raw}")


def parse_amount(raw: str) -> Decimal:
    if raw is None:
        raise ValueError("missing amount")
    value = raw.strip()
    if value == "":
        raise ValueError("empty amount")
    try:
        return Decimal(value)
    except InvalidOperation as exc:
        raise ValueError(f"invalid amount: {raw}") from exc


def coalesce(*values: str) -> str:
    for value in values:
        if value:
            return value
    return ""


def main() -> int:
    args = parse_args()

    if not args.contact and not args.contact_from:
        warn("--contact or --contact-from not provided; using 'Unknown' contact")

    rows: list[dict[str, str]] = []
    try:
        with open(args.csv_path, newline="") as handle:
            reader = csv.DictReader(handle)
            for row in reader:
                rows.append(row)
    except FileNotFoundError:
        print(f"error: file not found: {args.csv_path}", file=sys.stderr)
        return 2

    if not rows:
        print("error: no rows found in CSV", file=sys.stderr)
        return 2

    currencies = {row.get("CurrencyPrimary", "").strip() for row in rows if row}
    currencies.discard("")
    if len(currencies) > 1:
        warn(
            "multiple currencies detected in file; Xero BankTransactions require a bank account per currency"
        )

    errors = 0
    transactions: list[dict[str, object]] = []

    for idx, row in enumerate(rows, start=2):  # 1-based header, so data starts on line 2
        try:
            date_raw = row.get(args.date_column, "")
            if not date_raw:
                date_raw = row.get("Date/Time", "")
            date = parse_date(date_raw)

            amount = parse_amount(row.get(args.amount_column, ""))
            if amount == 0:
                warn(f"row {idx}: amount is zero; skipping")
                continue

            if args.force_type:
                txn_type = args.force_type
            else:
                txn_type = "RECEIVE" if amount > 0 else "SPEND"

            description = row.get(args.description_column, "").strip()
            if not description:
                description = row.get(args.type_column, "").strip()

            reference = row.get(args.reference_column, "").strip()
            if not reference:
                reference = row.get("TradeID", "").strip()

            contact_name = ""
            if args.contact_from:
                contact_name = row.get(args.contact_from, "").strip()
            contact_name = coalesce(contact_name, args.contact or "", "Unknown")

            line_item = {
                "Description": description,
                "Quantity": 1,
                "UnitAmount": float(abs(amount)),
                "AccountCode": args.account_code,
            }

            txn: dict[str, object] = {
                "Type": txn_type,
                "Contact": {"Name": contact_name},
                "Date": date,
                "LineItems": [line_item],
            }

            if reference:
                txn["Reference"] = reference
            if args.bank_account_code:
                txn["BankAccount"] = {"Code": args.bank_account_code}

            transactions.append(txn)
        except ValueError as exc:
            errors += 1
            warn(f"row {idx}: {exc}")
            if args.strict:
                return 2

    if not transactions:
        print("error: no valid transactions parsed", file=sys.stderr)
        return 2

    payload = {"BankTransactions": transactions}

    if args.pretty:
        json.dump(payload, sys.stdout, indent=2)
        sys.stdout.write("\n")
    else:
        json.dump(payload, sys.stdout)
        sys.stdout.write("\n")

    if errors and args.strict:
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
