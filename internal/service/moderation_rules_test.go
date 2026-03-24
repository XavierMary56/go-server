package service

import "testing"

func TestDetectStrongAdOrContact(t *testing.T) {
	cases := []struct {
		name    string
		content string
		wantHit bool
	}{
		{name: "wechat diversion", content: "加V test_video_01 领更多资源", wantHit: true},
		{name: "tg diversion", content: "TG 房间 testvideo05，外站继续聊", wantHit: true},
		{name: "offsite trade", content: "站外私聊交易，外站价更低，编号 tv03", wantHit: true},
		{name: "profile diversion", content: "看我头像，主页有方式，细聊", wantHit: true},
		{name: "adult only", content: "这段描写很露骨，成人味很重，但只是讨论内容本身。", wantHit: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, hit := detectStrongAdOrContact(tc.content)
			if hit != tc.wantHit {
				t.Fatalf("detectStrongAdOrContact(%q) = %v, want %v", tc.content, hit, tc.wantHit)
			}
		})
	}
}

func TestDetectWeakDrainSignal(t *testing.T) {
	cases := []struct {
		name    string
		content string
		wantHit bool
	}{
		{name: "weak drain", content: "有兴趣可以私聊我细聊", wantHit: true},
		{name: "normal adult", content: "这段描写很露骨，成人味很重，但只是讨论内容本身。", wantHit: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, hit := detectWeakDrainSignal(tc.content)
			if hit != tc.wantHit {
				t.Fatalf("detectWeakDrainSignal(%q) = %v, want %v", tc.content, hit, tc.wantHit)
			}
		})
	}
}

func TestApplyDeterministicDecisionWithAuditContentPrefix(t *testing.T) {
	result := applyDeterministicDecision("主内容：\n加V test_video_01 领更多资源", "standard")
	if result == nil {
		t.Fatal("expected rule-engine result, got nil")
	}
	if result.ModelUsed != "rule-engine" {
		t.Fatalf("unexpected model used: %s", result.ModelUsed)
	}
	if result.Verdict != "rejected" {
		t.Fatalf("unexpected verdict: %s", result.Verdict)
	}
}
