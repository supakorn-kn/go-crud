package errors

const (
	CurrentPageInvalidErrorCode   = 200_001
	ObjectIDNotFoundErrorCode     = 200_002
	DuplicatedObjectIDErrorCode   = 200_003
	MatchTypeInvalidErrorCode     = 200_004
	SortListInvalidErrorCode      = 200_005
	MatchKeyDuplicatedErrorCode   = 200_006
	MatchValueInvalidErrorCode    = 200_007
	DataAlreadyInUsedErrorCode    = 200_008
	DataValidationFailedErrorCode = 200_009
)

// CurrentPageInvalidError indicates user gives invalid current page when searching items
var CurrentPageInvalidError = new(CurrentPageInvalidErrorCode, ResponseError, "CurrentPageInvalid", "Current page can be only positive integer")

// ObjectIDNotFoundError indicates user gives invalid item ID
var ObjectIDNotFoundError = new(ObjectIDNotFoundErrorCode, ResponseError, "ObjectIDNotFound", "Item with ID %s is not exist")

// DuplicatedObjectIDError indicates user create item using item ID that already in used
var DuplicatedObjectIDError = new(DuplicatedObjectIDErrorCode, ResponseError, "DuplicatedObjectID", "item ID %s is already used")

// MatchTypeInvalidError indicates user give invalid or unsupported match type when user search items
var MatchTypeInvalidError = new(MatchTypeInvalidErrorCode, ResponseError, "MatchTypeInvalid", "Match type %s is invalid or unsupported")

// MatchValueInvalidError indicates internal error when server try to create match query using mismatch match value type
var MatchValueInvalidError = new(MatchValueInvalidErrorCode, InternalServerError, "MatchValueInvalid", "Given Match value's type %v is invalid or unsupported in Match type %s")

// MatchTypeInvalidError indicates internal error when server create sort query but list to sort is empty
var SortListInvalidError = new(SortListInvalidErrorCode, InternalServerError, "SortListInvalid", "Sort list length should not have been zero")

// MatchKeyDuplicatedError indicates internal error when server create match query but there's more than one same key to do matching
var MatchKeyDuplicatedError = new(MatchKeyDuplicatedErrorCode, InternalServerError, "MatchKeyDuplicated", "Match key %s is duplicated")

// DataAlreadyInUsedError indicates user give data to insert but there's one used the data which cannot be duplicated
var DataAlreadyInUsedError = new(DataAlreadyInUsedErrorCode, ResponseError, "DataAlreadyInUsed", "Given data is already in used")

// DataValidationFailedError indicates user give data to insert but it's fail on validation
var DataValidationFailedError = new(DataValidationFailedErrorCode, ResponseError, "DataValidationFailed", "Given data is invalid or cannot be used")
