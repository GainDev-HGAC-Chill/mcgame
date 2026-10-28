// Guild RPC registration and callback wire contracts.
package guild

import (
	"context"
	"errors"
	"reflect"
	"testing"

	domainguild "mcgame-server/internal/domain/guild"
	"mcgame-server/internal/infrastructure/config"
	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

type guildCallbackCall struct {
	method string
	args   []interface{}
}

type guildCallbackTransport struct {
	calls []guildCallbackCall
}

func (t *guildCallbackTransport) WriteCommand(_ context.Context, method string, args []interface{}) error {
	t.calls = append(t.calls, guildCallbackCall{method: method, args: args})
	return nil
}

func (t *guildCallbackTransport) WriteCommandSync(ctx context.Context, method string, args []interface{}) error {
	return t.WriteCommand(ctx, method, args)
}

func (t *guildCallbackTransport) WriteResult(context.Context, int64, interface{}) error {
	return nil
}

func (t *guildCallbackTransport) WriteError(context.Context, int64, error) error {
	return nil
}

func (t *guildCallbackTransport) Kind() string {
	return "test"
}

func (t *guildCallbackTransport) Close() error {
	return nil
}

func TestGuildRegistersExactRPCSurface(t *testing.T) {
	methods := []string{
		"initViewGuildP", "getGuildMemberList", "initGuildList", "clickBuild", "execBuildFunc", "constructBuild",
		"finishCreate", "finishUpgrade", "resetConstruction", "createGuildBuildings", "getInitGuildSlot", "addGuild",
		"addGuildMember", "quitGuild", "confirmGuildApply", "refuseGuildMember", "kickGuildMember", "demiseTo",
		"updateGuildMember", "updateGuildRank", "updateGuildNotice", "changeGuildName", "delGuild", "guildLevelUp",
		"addGuildBankSlotNum", "addMaxGuildMemberNum", "getGuildPrivateSkillData", "contribMoney", "contribMaterial",
		"moveGuildItem", "dropGuildItem", "developSkill",
	}
	if len(methods) != 32 {
		t.Fatalf("guild RPC fixture count = %d, want 32", len(methods))
	}

	dispatcher := rtmp.NewRPCDispatcher(zap.NewNop())
	NewHandler(zap.NewNop()).RegisterHandlers(dispatcher)
	conn := rtmp.NewConnection(1, nil, dispatcher, zap.NewNop())
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			_, err := dispatcher.Dispatch(conn, method, nil)
			if errors.Is(err, pkgerrors.ErrMethodNotFound) {
				t.Fatalf("guild RPC %q is not registered", method)
			}
		})
	}
}

func TestGuildVisibleCallbackContracts(t *testing.T) {
	logger := zap.NewNop()
	dispatcher := rtmp.NewRPCDispatcher(logger)
	server := rtmp.NewServer(config.RTMPConfig{}, logger, dispatcher)
	conn := server.AllocateConnection()
	conn.SetCharacter("7")
	conn.SetSceneInfo(49, 7)
	transport := &guildCallbackTransport{}
	conn.SetTransport(transport)
	h := NewHandler(logger)
	h.SetRTMPServer(server)
	guildItem := &domainguild.Guild{ID: 3, Members: []*domainguild.GuildMember{{CharacterID: 7}}}
	rankData := map[string]interface{}{"name2": "Phó bang"}

	h.broadcastGuildRank(guildItem, rankData)
	h.broadcastGuildProperty(guildItem, map[string]interface{}{"level": 2})
	h.broadcastGuildNumber(guildItem, map[string]interface{}{"type": 3, "num": 10})
	h.broadcastGuildInfo(guildItem, map[string]interface{}{"id": "3"})

	wantMethods := []string{"onUpdateGuildRank", "onGuildPropertyRefresh", "onUpdateGuildNumProp", "updateGuildInfo"}
	if len(transport.calls) != len(wantMethods) {
		t.Fatalf("callback count = %d, want %d", len(transport.calls), len(wantMethods))
	}
	for index, method := range wantMethods {
		if transport.calls[index].method != method {
			t.Fatalf("callback[%d] = %q, want %q", index, transport.calls[index].method, method)
		}
	}
	if len(transport.calls[0].args) != 1 {
		t.Fatalf("onUpdateGuildRank arg count = %d, want 1", len(transport.calls[0].args))
	}
	wrapper, ok := transport.calls[0].args[0].(map[string]interface{})
	if !ok || wrapper["gid"] != int64(3) || !reflect.DeepEqual(wrapper["rankData"], rankData) {
		t.Fatalf("onUpdateGuildRank args = %#v", transport.calls[0].args)
	}
}

func TestGuildApprovalRowIdentityAndApplicantCallbackOrder(t *testing.T) {
	logger := zap.NewNop()
	dispatcher := rtmp.NewRPCDispatcher(logger)
	server := rtmp.NewServer(config.RTMPConfig{}, logger, dispatcher)
	leaderConn := server.AllocateConnection()
	leaderConn.SetCharacter("7")
	leaderConn.SetSceneInfo(49, 7)
	leaderTransport := &guildCallbackTransport{}
	leaderConn.SetTransport(leaderTransport)
	applicantConn := server.AllocateConnection()
	applicantConn.SetCharacter("9")
	applicantConn.SetSceneInfo(49, 9)
	applicantTransport := &guildCallbackTransport{}
	applicantConn.SetTransport(applicantTransport)
	h := NewHandler(logger)
	h.SetRTMPServer(server)
	application := &domainguild.GuildApplication{ID: 41, GuildID: 3, CharacterID: 9}
	member := &domainguild.GuildMember{ID: 77, GuildID: 3, CharacterID: 9, Rank: domainguild.RankMember, ContributionNormal: 90, ContributionDonate: 12}
	guildItem := &domainguild.Guild{ID: 3, Members: []*domainguild.GuildMember{{CharacterID: 7}, member}}
	payload := h.guildApprovalPayload(application, member)

	h.notifyGuildApproval(guildItem, application, member, payload)
	panelResponse := map[string]interface{}{"memberList": []interface{}{member.ToDTO()}}
	if err := h.sendGuildPanelRefresh(leaderConn, panelResponse); err != nil {
		t.Fatalf("send approver panel refresh: %v", err)
	}

	if payload["tableId"] != int64(41) {
		t.Fatalf("approval tableId = %#v, want application ID 41", payload["tableId"])
	}
	gData, ok := payload["gData"].(map[string]interface{})
	if !ok || gData["tableId"] != int64(77) {
		t.Fatalf("approval gData = %#v, want committed member tableId 77", payload["gData"])
	}
	if len(leaderTransport.calls) != 2 || leaderTransport.calls[0].method != "onConfirmGuildApply" || leaderTransport.calls[1].method != "onInitViewGuildP" {
		t.Fatalf("leader callbacks = %#v", leaderTransport.calls)
	}
	if !reflect.DeepEqual(leaderTransport.calls[1].args, []interface{}{panelResponse}) {
		t.Fatalf("leader panel refresh args = %#v", leaderTransport.calls[1].args)
	}
	wantApplicantMethods := []string{"onConfirmGuildApply", "onUPP", "a"}
	if len(applicantTransport.calls) != len(wantApplicantMethods) {
		t.Fatalf("applicant callback count = %d, want %d", len(applicantTransport.calls), len(wantApplicantMethods))
	}
	for index, method := range wantApplicantMethods {
		if applicantTransport.calls[index].method != method {
			t.Fatalf("applicant callback[%d] = %q, want %q", index, applicantTransport.calls[index].method, method)
		}
	}
	contribution, ok := applicantTransport.calls[1].args[0].(map[string]interface{})
	if !ok || contribution["guildContrib"] != int64(90) || contribution["donateContrib"] != int64(12) {
		t.Fatalf("applicant contribution callback = %#v", applicantTransport.calls[1].args)
	}

	applicantTransport.calls = nil
	h.sendGuildApprovalToApplicant(application.CharacterID, member, payload)
	if len(applicantTransport.calls) != len(wantApplicantMethods) {
		t.Fatalf("fallback applicant callback count = %d, want %d", len(applicantTransport.calls), len(wantApplicantMethods))
	}
	for index, method := range wantApplicantMethods {
		if applicantTransport.calls[index].method != method {
			t.Fatalf("fallback applicant callback[%d] = %q, want %q", index, applicantTransport.calls[index].method, method)
		}
	}
}

var _ rtmp.Transport = (*guildCallbackTransport)(nil)
