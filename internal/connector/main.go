// main 패키지는 Discord 봇의 주요 기능을 포함합니다.
// 필요한 라이브러리들을 임포트합니다.
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

// Discord 명령어 및 UI 요소에 사용될 상수들을 정의합니다.
const (
	cmdAgent          = "agent"
	subCmdCreate      = "create"
	subCmdList        = "list"
	subCmdView        = "view"
	subCmdDelete      = "delete"
	subCmdEdit        = "edit"
	subCmdCall        = "call"
	prefixModalCreate = "modal_agent_create"
	prefixModalEdit   = "modal_agent_edit_"
	prefixButtonEdit  = "edit_agent_"
)

// Agent 구조체는 에이전트의 이름, 설명, 모델, 프롬프트 등 모든 정보를 담습니다.
type Agent struct {
	Name        string
	Description string
	Model       string
	Prompt      string
}

// agentsMutex는 agents 맵에 대한 동시성 접근을 제어합니다.
// agents 맵은 생성된 에이전트들을 이름으로 저장합니다.
// threadsMutex는 activeThreads 맵에 대한 동시성 접근을 제어합니다.
// activeThreads 맵은 현재 활성화된 스레드와 연결된 에이전트 이름을 저장합니다.
var (
	agentsMutex   sync.RWMutex
	agents        = make(map[string]*Agent)
	threadsMutex  sync.RWMutex
	activeThreads = make(map[string]string)
)

// main 함수는 봇의 시작점입니다.
// 환경 변수를 로드하고, Discord 세션을 생성하며, 이벤트 핸들러를 등록하고,
// 봇 연결을 열고, 종료 신호를 기다려 봇을 안전하게 종료합니다.
func main() {
	godotenv.Load()
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("Error: DISCORD_TOKEN environment variable not set.")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}
	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages

	dg.AddHandler(readyHandler)
	dg.AddHandler(interactionRouter)
	dg.AddHandler(messageCreateHandler)

	if err := dg.Open(); err != nil {
		log.Fatalf("Error opening connection: %v", err)
	}

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	dg.Close()
	fmt.Println("Bot gracefully shut down.")
}

// readyHandler는 봇이 Discord에 성공적으로 연결되었을 때 호출됩니다.
// 여기서 전역 애플리케이션 명령어를 등록합니다.
func readyHandler(s *discordgo.Session, r *discordgo.Ready) {
	log.Println("Bot is ready! Registering commands...")

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        cmdAgent,
			Description: "에이전트 관리 및 호출 명령어",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: subCmdCreate, Description: "새로운 에이전트를 생성합니다."},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: subCmdList, Description: "생성된 에이전트 목록을 봅니다."},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: subCmdView, Description: "특정 에이전트의 상세 정보를 봅니다.", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "name", Description: "정보를 볼 에이전트의 이름", Required: true, Autocomplete: true}}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: subCmdDelete, Description: "특정 에이전트를 삭제합니다.", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "name", Description: "삭제할 에이전트의 이름", Required: true, Autocomplete: true}}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: subCmdEdit, Description: "특정 에이전트의 정보를 수정합니다.", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "name", Description: "수정할 에이전트의 이름", Required: true, Autocomplete: true}}},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: subCmdCall, Description: "에이전트와의 대화 스레드를 시작합니다.", Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "name", Description: "호출할 에이전트의 이름", Required: true, Autocomplete: true}}},
			},
		},
	}

	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", commands)
	if err != nil {
		log.Fatalf("Could not register commands: %v", err)
	}
	log.Println("Successfully registered commands.")
}

// interactionRouter는 Discord에서 발생하는 다양한 상호작용(슬래시 명령어, 버튼 클릭, 모달 제출, 자동 완성)을
// 유형에 따라 적절한 핸들러 함수로 라우팅합니다.
func interactionRouter(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		handleSlashCommand(s, i)
	case discordgo.InteractionMessageComponent:
		handleButton(s, i)
	case discordgo.InteractionModalSubmit:
		handleModal(s, i)
	case discordgo.InteractionApplicationCommandAutocomplete:
		handleAutocomplete(s, i)
	}
}

// messageCreateHandler는 새로운 메시지가 생성될 때 호출됩니다.
// 봇 자신의 메시지는 무시하고, 활성화된 에이전트 스레드에 메시지가 전송된 경우
// 해당 에이전트에게 메시지를 전달합니다.
func messageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	threadsMutex.RLock()
	agentName, ok := activeThreads[m.ChannelID]
	threadsMutex.RUnlock()

	if ok {
		agentsMutex.RLock()
		agent, agentOk := agents[agentName]
		agentsMutex.RUnlock()

		if !agentOk {
			s.ChannelMessageSend(m.ChannelID, "오류: 이 스레드에 연결된 에이전트를 찾을 수 없습니다.")
			return
		}
		callAgentInThread(s, m.Message, agent)
	}
}

// handleSlashCommand는 '/agent' 슬래시 명령어를 처리합니다.
// 하위 명령어(생성, 목록, 보기, 삭제, 수정, 호출)에 따라 적절한 함수를 호출합니다.
func handleSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ApplicationCommandData().Name != cmdAgent {
		return
	}
	subCommand := i.ApplicationCommandData().Options[0]
	switch subCommand.Name {
	case subCmdCreate:
		showCreateOrEditModal(s, i, "", nil)
	case subCmdList:
		showAgentList(s, i)
	case subCmdView:
		showAgentDetails(s, i, subCommand.Options[0].StringValue())
	case subCmdDelete:
		deleteAgent(s, i, subCommand.Options[0].StringValue())
	case subCmdEdit:
		showEditUI(s, i, subCommand.Options[0].StringValue())
	case subCmdCall:
		startAgentThread(s, i, subCommand.Options[0].StringValue())
	}
}

// handleButton은 버튼 클릭 상호작용을 처리합니다.
// 에이전트 수정 버튼이 클릭된 경우, 해당 에이전트의 수정 모달을 표시합니다.
func handleButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	if strings.HasPrefix(customID, prefixButtonEdit) {
		agentName := strings.TrimPrefix(customID, prefixButtonEdit)
		agentsMutex.RLock()
		agent, ok := agents[agentName]
		agentsMutex.RUnlock()
		if ok {
			showCreateOrEditModal(s, i, agentName, agent)
		}
	}
}

// handleModal은 모달 제출 상호작용을 처리합니다.
// 에이전트 생성 또는 수정 모달에서 제출된 데이터를 파싱하여 에이전트 정보를 업데이트합니다.
func handleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	data := i.ModalSubmitData().Components
	name := data[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	desc := data[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	model := data[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	prompt := data[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	agentsMutex.Lock()
	defer agentsMutex.Unlock()

	switch {
	case customID == prefixModalCreate:
		if _, ok := agents[name]; ok {
			respondEphemeral(s, i, fmt.Sprintf("오류: 에이전트 '**%s**'은(는) 이미 존재해요.", name))
			return
		}
		agents[name] = &Agent{Name: name, Description: desc, Model: model, Prompt: prompt}
		respondEphemeral(s, i, fmt.Sprintf("에이전트 '**%s**'이(가) 성공적으로 생성되었어요!", name))

	case strings.HasPrefix(customID, prefixModalEdit):
		originalName := strings.TrimPrefix(customID, prefixModalEdit)
		if originalName != name {
			if _, ok := agents[name]; ok {
				respondEphemeral(s, i, fmt.Sprintf("오류: 변경하려는 이름 '**%s**'은(는) 이미 다른 에이전트가 사용 중이에요.", name))
				return
			}
			delete(agents, originalName)
		}
		agents[name] = &Agent{Name: name, Description: desc, Model: model, Prompt: prompt}
		respondEphemeral(s, i, fmt.Sprintf("에이전트 '**%s**'의 정보가 성공적으로 수정되었어요!", name))
	}
}

// handleAutocomplete는 자동 완성 상호작용을 처리합니다.
// 에이전트 이름 입력 시 현재 생성된 에이전트 목록을 기반으로 자동 완성 제안을 제공합니다.
func handleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options[0].Options[0]
	if options.Focused {
		agentsMutex.RLock()
		defer agentsMutex.RUnlock()
		var choices []*discordgo.ApplicationCommandOptionChoice
		for name := range agents {
			if strings.HasPrefix(strings.ToLower(name), strings.ToLower(options.StringValue())) {
				choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: name, Value: name})
			}
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionApplicationCommandAutocompleteResult, Data: &discordgo.InteractionResponseData{Choices: choices}})
	}
}

// respondEphemeral은 사용자에게만 보이는 임시 메시지를 전송합니다.
func respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral}})
}

// startAgentThread는 지정된 에이전트와의 새로운 대화 스레드를 시작합니다.
// 스레드를 생성하고, 해당 스레드에 에이전트 정보를 담은 시작 메시지를 보냅니다.
func startAgentThread(s *discordgo.Session, i *discordgo.InteractionCreate, agentName string) {
	agentsMutex.RLock()
	agent, ok := agents[agentName]
	agentsMutex.RUnlock()
	if !ok {
		respondEphemeral(s, i, fmt.Sprintf("에이전트 '**%s**'을(를) 찾을 수 없어요.", agentName))
		return
	}

	// 먼저 사용자에게만 보이는 확인 메시지로 응답해요.
	respondEphemeral(s, i, fmt.Sprintf("'**%s**'와의 대화 스레드를 생성 중...", agentName))

	// 채널에 공개 스레드를 생성해요.
	thread, err := s.ThreadStart(i.ChannelID, fmt.Sprintf("[%s] 대화방", agent.Name), discordgo.ChannelTypeGuildPublicThread, 60)
	if err != nil {
		log.Printf("Failed to create thread: %v", err)
		return
	}

	threadsMutex.Lock()
	activeThreads[thread.ID] = agent.Name
	threadsMutex.Unlock()

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("'%s'와의 대화 시작", agent.Name),
		Description: "이 스레드에 메시지를 입력하여 대화를 시작하세요.",
		Color:       0x33cc33, // Green
		Fields: []*discordgo.MessageEmbedField{
			{Name: "에이전트 모델", Value: agent.Model, Inline: true},
			{Name: "역할 정의 (프롬프트)", Value: fmt.Sprintf("```\n%s\n```", agent.Prompt), Inline: false},
		},
	}
	s.ChannelMessageSendEmbed(thread.ID, embed)
}

// callAgentInThread는 활성화된 에이전트 스레드 내에서 메시지를 처리합니다.
// 현재는 간단한 테스트 응답을 제공하며, 실제 LLM 호출 로직이 구현될 예정입니다.
func callAgentInThread(s *discordgo.Session, m *discordgo.Message, agent *Agent) {
	// 테스트: "안녕!"이라고 하면 "안녕하세요!"라고 대답
	if m.Content == "안녕!" {
		s.ChannelMessageSend(m.ChannelID, "안녕하세요!")
		return
	}

	// 실제 LLM 호출 등은 여기에 구현해요.
	// 지금은 받은 메시지를 확인하는 응답만 보내요.
	embed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{Name: m.Author.Username, IconURL: m.Author.AvatarURL("")},
		Description: m.Content,
		Color:       0x0099ff, // Blue
		Footer:      &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("'%s'에게 전달됨 (실행 기능은 미구현)", agent.Name)},
	}
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// showAgentList는 현재 등록된 모든 에이전트의 목록을 Discord에 표시합니다.
// 에이전트가 없는 경우 해당 메시지를 사용자에게 보냅니다.
func showAgentList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	agentsMutex.RLock()
	defer agentsMutex.RUnlock()
	if len(agents) == 0 {
		respondEphemeral(s, i, "생성된 에이전트가 아직 없어요. `/agent create`로 먼저 생성해주세요!")
		return
	}
	fields := []*discordgo.MessageEmbedField{}
	for name, agent := range agents {
		fields = append(fields, &discordgo.MessageEmbedField{Name: name, Value: agent.Description, Inline: false})
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{{
		Title: "생성된 에이전트 목록", Fields: fields, Color: 0x0099ff,
	}}}})
}

// showAgentDetails는 특정 에이전트의 상세 정보를 Discord에 표시합니다.
// 에이전트를 찾을 수 없는 경우 오류 메시지를 보냅니다.
func showAgentDetails(s *discordgo.Session, i *discordgo.InteractionCreate, name string) {
	agentsMutex.RLock()
	defer agentsMutex.RUnlock()
	agent, ok := agents[name]
	if !ok {
		respondEphemeral(s, i, fmt.Sprintf("에이전트 '**%s**'을(를) 찾을 수 없어요.", name))
		return
	}
	embed := &discordgo.MessageEmbed{
		Title: "에이전트 상세 정보: " + agent.Name, Color: 0x0099ff,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "설명", Value: agent.Description},
			{Name: "모델", Value: agent.Model, Inline: true},
			{Name: "역할 정의 (프롬프트)", Value: fmt.Sprintf("```\n%s\n```", agent.Prompt)},
			{Name: "실행한 작업 목록", Value: "(아직 구현되지 않은 기능이에요)"},
		},
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}}})
}

// deleteAgent는 지정된 이름의 에이전트를 삭제합니다.
// 에이전트를 찾을 수 없는 경우 오류 메시지를 보내고, 삭제 성공 시 관련 활성 스레드 정보도 정리합니다.
func deleteAgent(s *discordgo.Session, i *discordgo.InteractionCreate, name string) {
	agentsMutex.Lock()
	defer agentsMutex.Unlock()
	if _, ok := agents[name]; !ok {
		respondEphemeral(s, i, fmt.Sprintf("에이전트 '**%s**'을(를) 찾을 수 없어요.", name))
		return
	}
	delete(agents, name)

	threadsMutex.Lock()
	defer threadsMutex.Unlock()
	for threadID, agentName := range activeThreads {
		if agentName == name {
			delete(activeThreads, threadID)
		}
	}
	respondEphemeral(s, i, fmt.Sprintf("에이전트 '**%s**'이(가) 성공적으로 삭제되었어요.", name))
}

// showEditUI는 특정 에이전트의 현재 정보를 임베드 메시지로 표시하고,
// 정보를 수정할 수 있는 모달을 열기 위한 버튼을 제공합니다.
func showEditUI(s *discordgo.Session, i *discordgo.InteractionCreate, name string) {
	agentsMutex.RLock()
	defer agentsMutex.RUnlock()
	agent, ok := agents[name]
	if !ok {
		respondEphemeral(s, i, fmt.Sprintf("에이전트 '**%s**'을(를) 찾을 수 없어요.", name))
		return
	}
	embed := &discordgo.MessageEmbed{
		Title: "에이전트 수정: " + agent.Name, Description: "아래는 현재 정보예요. 수정하려면 버튼을 눌러주세요.", Color: 0xffaa00,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "설명", Value: agent.Description},
			{Name: "모델", Value: agent.Model, Inline: true},
			{Name: "역할 정의 (프롬프트)", Value: fmt.Sprintf("```\n%s\n```", agent.Prompt)},
		},
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "정보 수정하기", Style: discordgo.PrimaryButton, CustomID: prefixButtonEdit + name},
		}}},
	}})
}

// showCreateOrEditModal은 에이전트를 생성하거나 기존 에이전트의 정보를 수정하기 위한 모달을 동적으로 생성하고 표시합니다.
// agent 매개변수가 nil이 아니면 수정 모드로 작동합니다.
func showCreateOrEditModal(s *discordgo.Session, i *discordgo.InteractionCreate, originalName string, agent *Agent) {
	modalTitle := "새로운 에이전트 생성"
	customID := prefixModalCreate
	name, desc, model, prompt := "", "", "", ""

	if agent != nil { // 수정 모드일 경우
		modalTitle = "에이전트 정보 수정"
		customID = prefixModalEdit + originalName
		name, desc, model, prompt = agent.Name, agent.Description, agent.Model, agent.Prompt
	}

	modal := &discordgo.InteractionResponseData{
		CustomID: customID, Title: modalTitle,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "name", Label: "이름", Style: discordgo.TextInputShort, Required: true, Value: name}}},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "desc", Label: "설명", Style: discordgo.TextInputParagraph, Required: true, Value: desc}}},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "model", Label: "모델", Style: discordgo.TextInputShort, Required: true, Value: model}}},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "prompt", Label: "역할 정의 (프롬프트)", Style: discordgo.TextInputParagraph, Required: true, Value: prompt}}},
		},
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: modal})
}
