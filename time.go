package meg

import (
	"fmt"
	"time"
)

func TimeAgo(t time.Time) string {
	now := time.Now()
	future := t.After(now)

	var duration time.Duration
	if future {
		duration = t.Sub(now)
	} else {
		duration = now.Sub(t)
	}

	seconds := int(duration.Seconds())
	minutes := int(duration.Minutes())
	hours := int(duration.Hours())
	days := hours / 24
	weeks := days / 7
	months := days / 30
	years := days / 365

	if future {
		switch {
		case seconds < 10:
			return "через пару секунд"
		case seconds < 60:
			return "менее чем через минуту"
		case minutes == 1:
			return "через минуту"
		case minutes < 5:
			return fmt.Sprintf("через %d минуты", minutes)
		case minutes < 60:
			return fmt.Sprintf("через %d минут", minutes)
		case hours == 1:
			return "через час"
		case hours < 5:
			return fmt.Sprintf("через %d часа", hours)
		case hours < 24:
			return fmt.Sprintf("через %d часов", hours)
		case days == 1:
			return "завтра"
		case days < 5:
			return fmt.Sprintf("через %d дня", days)
		case days < 7:
			return fmt.Sprintf("через %d дней", days)
		case weeks == 1:
			return "через неделю"
		case weeks < 4:
			return fmt.Sprintf("через %d недели", weeks)
		case months == 1:
			return "через месяц"
		case months < 12:
			return fmt.Sprintf("через %d месяцев", months)
		case years == 1:
			return "через год"
		case years < 5:
			return fmt.Sprintf("через %d года", years)
		default:
			return "через несколько лет"
		}
	}

	switch {
	case seconds < 10:
		return "только что"
	case seconds < 60:
		return "менее минуты назад"
	case minutes == 1:
		return "минуту назад"
	case minutes < 5:
		return fmt.Sprintf("%d минуты назад", minutes)
	case minutes < 60:
		return fmt.Sprintf("%d минут назад", minutes)
	case hours == 1:
		return "час назад"
	case hours < 5:
		return fmt.Sprintf("%d часа назад", hours)
	case hours < 24:
		return fmt.Sprintf("%d часов назад", hours)
	case days == 1:
		return "вчера"
	case days < 5:
		return fmt.Sprintf("%d дня назад", days)
	case days < 7:
		return fmt.Sprintf("%d дней назад", days)
	case weeks == 1:
		return "неделю назад"
	case weeks < 4:
		return fmt.Sprintf("%d недели назад", weeks)
	case months == 1:
		return "месяц назад"
	case months < 12:
		return fmt.Sprintf("%d месяцев назад", months)
	case years == 1:
		return "год назад"
	case years < 5:
		return fmt.Sprintf("%d года назад", years)
	default:
		return "несколько лет назад"
	}
}
