package handler

// В реальной системе здесь была бы авторизация и хранение пользователя в контексте.
// Для лабораторной работы пользователь-создатель и модератор заданы константами,
// а доступ к ним осуществляется через функции-одиночки (singleton-style).

const (
	demoCreatorUserID   = 1
	demoModeratorUserID = 2
)

// CurrentUserID возвращает идентификатор текущего пользователя-создателя.
func CurrentUserID() int {
	return demoCreatorUserID
}

// CurrentModeratorID возвращает идентификатор пользователя-модератора.
func CurrentModeratorID() int {
	return demoModeratorUserID
}

