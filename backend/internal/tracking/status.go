package tracking

/*
Function:	ValidStatus
Purpose:	whether a status can be saved on a list entry. "wishlist" is still in the database's enum
but isn't a status any more: the wishlist is kept apart (user_wishlist) so wishlisting a title
never changes where the user says they are with it
Params:
  - status: the status asked for
*/
func ValidStatus(status string) bool {
	switch status {
	case "current", "planned", "completed", "dropped":
		return true
	}
	return false
}
