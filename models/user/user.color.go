package user

type AvatarColor string

const (
	ColorPrimary AvatarColor = "primary"
	ColorPink    AvatarColor = "pink"
	ColorRed     AvatarColor = "red"
	ColorYellow  AvatarColor = "yellow"
	ColorBlue    AvatarColor = "blue"
	ColorGreen   AvatarColor = "green"
	ColorPurple  AvatarColor = "purple"
	ColorOrange  AvatarColor = "orange"
	ColorGray    AvatarColor = "gray"
	ColorAmber   AvatarColor = "amber"
)

var colors = []string{
	string(ColorPrimary),
	string(ColorPink),
	string(ColorRed),
	string(ColorYellow),
	string(ColorBlue),
	string(ColorGreen),
	string(ColorPurple),
	string(ColorOrange),
	string(ColorGray),
	string(ColorAmber),
}

func (AvatarColor) Enum() []string {
	return colors
}

func ColorFromEmail(email string) AvatarColor {
	sum := 0
	for _, r := range email {
		sum += int(r)
	}

	return AvatarColor(colors[sum%len(colors)])
}
