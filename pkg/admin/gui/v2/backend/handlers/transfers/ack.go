package transfers

import (
	"fmt"
	"time"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// The acknowledgement of each transfer in the list: a column "ACK"
// after the end, an icon and one word, the detail in the tooltip of the
// kit. The keys of a page are read in one query, never one per row; the
// case comes from the model, the same rules as the filter, the REST API and
// the command line.

// ackView is the cell of the acknowledgement of a transfer.
type ackView struct {
	Icon  string // the shape of the case, an icon of the kit
	Label string // one word; "no acknowledgement" for the aids when the cell shows "—"
	Title string // the detail: the moments, the limit, the partner
	None  bool   // no acknowledgement: "—"
}

// ackKeys are the keys of the acknowledgement family of the registry.
func ackKeys() []any {
	var keys []any

	for family, key := range model.ReservedInfoKeys.Iter() {
		if family == model.InfoFamilyAck {
			keys = append(keys, key.Name)
		}
	}

	return keys
}

// ackInfos reads in one query the acknowledgement keys of the transfers of a
// page, by transfer: the normalized view of the information names the owner
// of a row whether the transfer runs or is over.
func (h *Handlers) ackInfos(transfers []*model.NormalizedTransferView) (map[int64]map[string]any, error) {
	byTransfer := map[int64]map[string]any{}
	if len(transfers) == 0 {
		return byTransfer, nil
	}

	ids := make([]any, 0, len(transfers))
	for _, trans := range transfers {
		ids = append(ids, trans.ID)
	}

	var infos model.NormalizedTransferInfos
	if err := h.db.Select(&infos).In("owner_id", ids...).In("name", ackKeys()...).Run(); err != nil {
		return nil, fmt.Errorf("failed to read the acknowledgements of the transfers: %w", err)
	}

	for _, info := range infos {
		if byTransfer[info.OwnerID] == nil {
			byTransfer[info.OwnerID] = map[string]any{}
		}

		byTransfer[info.OwnerID][info.Name] = info.Value
	}

	return byTransfer, nil
}

// ackIcon is the shape of a case: a clock for the wait, a check for the
// acknowledgement received, a cross for the one missed, an envelope for the
// one sent.
func ackIcon(ackCase model.AckCase) string {
	switch ackCase {
	case model.AckExpected:
		return "clock"
	case model.AckReceived:
		return "check"
	case model.AckMissed:
		return "xmark"
	case model.AckSent:
		return "envelope"
	case model.AckNone:
		return ""
	default:
		return ""
	}
}

// ackLabelKey is the key of the word of a case, spelled out for the
// catalogue test.
func ackLabelKey(ackCase model.AckCase) string {
	switch ackCase {
	case model.AckExpected:
		return "transfers.ack.expected"
	case model.AckReceived:
		return "transfers.ack.received"
	case model.AckMissed:
		return "transfers.ack.missed"
	case model.AckSent:
		return "transfers.ack.sent"
	case model.AckNone:
		return "transfers.ack.none"
	default:
		return "transfers.ack.none"
	}
}

// ackMoment reads a moment of the information: the gateway writes a time,
// read back as its RFC 3339 text.
func ackMoment(value any) (time.Time, bool) {
	switch v := value.(type) {
	case time.Time:
		return v, !v.IsZero()
	case string:
		at, err := time.Parse(time.RFC3339Nano, v)

		return at, err == nil
	default:
		return time.Time{}, false
	}
}

// ackLimit is how long the gateway waits for the acknowledgement of a
// transfer: the limit of the transfer, else the one of the pipeline.
func ackLimit(info map[string]any) time.Duration {
	if stored, ok := info[model.InfoAckWaitTimeout].(string); ok {
		if limit, err := time.ParseDuration(stored); err == nil && limit > 0 {
			return limit
		}
	}

	return model.AckDefaultWait
}

// ackCell lays out the acknowledgement of a transfer from its keys: one
// word, the detail in the tooltip; an acknowledgement sent by the gateway
// adds its sentence to the tooltip of another case (a relay).
func ackCell(translate gui.Translator, trans *model.NormalizedTransferView, info map[string]any) ackView {
	ackCase := model.AckOf(trans.Status, info)
	if ackCase == model.AckNone {
		none := translate("transfers.ack.none")

		return ackView{None: true, Label: none, Title: none}
	}

	limit := ackLimit(info)
	limitText := protoform.FormatDuration(translate, limit.String())
	since, waiting := ackMoment(info[model.InfoAckWaitSince])

	var title string

	switch ackCase {
	case model.AckExpected:
		if waiting {
			title = translate("transfers.ack_tip.waiting", "until", dateTime(translate, since.Add(limit)),
				"limit", limitText, "since", dateTime(translate, since))
		} else {
			title = translate("transfers.ack_tip.expected", "limit", limitText)
		}
	case model.AckReceived:
		at, _ := ackMoment(info[model.InfoAckReceivedOn])
		title = translate("transfers.ack_tip.received", "at", dateTime(translate, at))
	case model.AckMissed:
		if waiting {
			title = translate("transfers.ack_tip.missed_since", "limit", limitText, "since", dateTime(translate, since))
		} else {
			title = translate("transfers.ack_tip.missed", "limit", limitText)
		}
	case model.AckSent:
		title = ackSentSentence(translate, info)
	case model.AckNone:
		// Handled above: the cell shows "—".
	}

	if ackCase != model.AckSent && model.AckWasSent(info) {
		title += " · " + ackSentSentence(translate, info)
	}

	return ackView{Icon: ackIcon(ackCase), Label: translate(ackLabelKey(ackCase)), Title: title}
}

// ackSentSentence says when the gateway acknowledged the transfer, and to
// whom when the information names the partner.
func ackSentSentence(translate gui.Translator, info map[string]any) string {
	at, _ := ackMoment(info[model.InfoAckSentOn])
	partner, _ := info[model.InfoAckSentTo].(string)
	account, _ := info[model.InfoAckSentAs].(string)

	if partner == "" {
		return translate("transfers.ack_tip.sent_on", "at", dateTime(translate, at))
	}

	return translate("transfers.ack_tip.sent", "at", dateTime(translate, at), "partner", partner, "account", account)
}
