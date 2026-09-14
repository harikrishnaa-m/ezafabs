package pricing

import "math"

type Totals struct {
	SubTotal       float64 `json:"sub_total"`
	DiscountAmount float64 `json:"discount_amount"`
	TaxAmount      float64 `json:"tax_amount"`
	ShippingCharge float64 `json:"shipping_charge"`
	GrandTotal     float64 `json:"grand_total"`
}

// Calculate applies the single e-commerce total formula used by cart and checkout.
// Product prices are currently tax-inclusive and no e-commerce coupon/tax rules
// are configured yet, so discount and tax remain zero until those rules exist.
func Calculate(subTotal, shippingCharge float64) Totals {
	subTotal = round2(subTotal)
	shippingCharge = round2(shippingCharge)
	discountAmount := 0.0
	taxAmount := 0.0
	grandTotal := round2(subTotal - discountAmount + taxAmount + shippingCharge)

	return Totals{
		SubTotal:       subTotal,
		DiscountAmount: discountAmount,
		TaxAmount:      taxAmount,
		ShippingCharge: shippingCharge,
		GrandTotal:     grandTotal,
	}
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
