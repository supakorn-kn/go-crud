package errors

const (
	CurrentPageInvalidErrorCode = 200_001
	ObjectIDNotFoundErrorCode   = 200_002
	DuplicatedObjectIDErrorCode = 200_003
	MatchTypeInvalidErrorCode   = 200_004
)

// CurrentPageInvalidError indicates user gives invalid current page when searching items
var CurrentPageInvalidError = new(CurrentPageInvalidErrorCode, "CurrentPageInvalid", "Current page can be only positive integer")

// ObjectIDNotFoundError indicates user gives invalid item ID
var ObjectIDNotFoundError = new(ObjectIDNotFoundErrorCode, "ObjectIDNotFound", "Item with ID %s is not exist")

// DuplicatedObjectIDError indicates user create item using item ID that already in used
var DuplicatedObjectIDError = new(DuplicatedObjectIDErrorCode, "DuplicatedObjectID", "item ID %s is already used")

// MatchTypeInvalidError indicates user give invalid or unsupported match type when user search items
var MatchTypeInvalidError = new(MatchTypeInvalidErrorCode, "MatchTypeInvalid", "Match type %d is invalid or unsupported")
