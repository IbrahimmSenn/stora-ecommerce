// validation.go — one shopper-readable formatter for validator/v10 errors.
// The frontend renders these in an alert, so we want plain English, not field
// names and tag literals. Shared by every handler that validates a request.
package response

import "github.com/go-playground/validator/v10"

// FormatValidation turns validator/v10's typed errors into a single readable
// sentence. Unknown fields fall back to their struct name; unknown tags fall
// back to "is invalid".
func FormatValidation(ve validator.ValidationErrors) string {
	if len(ve) == 0 {
		return "please check your details and try again"
	}
	parts := make([]string, 0, len(ve))
	for _, fe := range ve {
		parts = append(parts, friendlyField(fe.Field())+" "+friendlyTag(fe))
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += "; " + p
	}
	return out
}

func friendlyField(name string) string {
	switch name {
	case "Email":
		return "Email"
	case "Phone":
		return "Phone"
	case "ShippingMethod":
		return "Shipping method"
	case "RecipientName":
		return "Recipient name"
	case "Line1":
		return "Address line 1"
	case "Line2":
		return "Address line 2"
	case "City":
		return "City"
	case "Region":
		return "State / region"
	case "PostalCode":
		return "Postal code"
	case "Country":
		return "Country"
	default:
		return name
	}
}

func friendlyTag(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return "is too short (min " + fe.Param() + " characters)"
	case "max":
		return "is too long (max " + fe.Param() + " characters)"
	case "len":
		return "must be exactly " + fe.Param() + " characters"
	case "alpha":
		return "must contain letters only"
	case "iso3166_1_alpha2":
		return "must be a valid ISO 3166-1 alpha-2 country code (e.g. US, GB, EE)"
	case "oneof":
		return "must be one of: " + fe.Param()
	default:
		return "is invalid"
	}
}
