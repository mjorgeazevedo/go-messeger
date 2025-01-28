package messeger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
)

const botToken = "7557709232:AAFX-J_Bd3QgLD5TQWa1z7vc7Z94CVhSfvk"
const urlTelegram = "https://api.telegram.org"

// const chatID = "7557709232"

type Update struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		MessageID int `json:"message_id"`
		From      struct {
			ID        int    `json:"id"`
			FirstName string `json:"first_name"`
			Username  string `json:"username"`
		} `json:"from"`
		Chat struct {
			ID int `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
}

type GetUpdatesResponse struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type WebhookInfo struct {
	Ok     bool `json:"ok"`
	Result struct {
		URL                  string `json:"url"`
		HasCustomCertificate bool   `json:"has_custom_certificate"`
		PendingUpdateCount   int    `json:"pending_update_count"`
	} `json:"result"`
}

func SetWebHook(urlWebHook string) error {
	apiURL := fmt.Sprintf(urlTelegram+"/bot%s/setWebhook?url=%s", botToken, urlWebHook)
	// Criar buffer para armazenar o body da requisição
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Adicionar URL do webhook
	_ = writer.WriteField("url", urlWebHook)

	certPath := "server.pem"

	// Adicionar certificado como arquivo
	file, err := os.Open(certPath)
	if err != nil {
		log.Fatal("Erro ao abrir o certificado:", err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("certificate", certPath)
	if err != nil {
		log.Fatal("Erro ao anexar o certificado:", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		log.Fatal("Erro ao copiar o certificado:", err)
	}

	// Fechar o multipart writer
	writer.Close()

	// Criar requisição HTTP
	req, err := http.NewRequest("POST", apiURL, &body)
	if err != nil {
		log.Fatal("Erro ao criar requisição:", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Enviar requisição
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Erro ao enviar requisição:", err)
	}
	defer resp.Body.Close()

	// Exibir resposta
	fmt.Println("Resposta do Telegram:", resp.Status)

	return nil
}

func DeleteWebHook() error {
	apiURL := fmt.Sprintf(urlTelegram+"/bot%s/deleteWebhook", botToken)
	slog.Debug(apiURL)
	// Criando os dados do POST
	data := url.Values{}

	// Fazendo a requisição HTTP
	resp, err := http.PostForm(apiURL, data)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	slog.Info("messeger :: DeleteWebHook :: status code -> " + fmt.Sprint(resp.StatusCode))

	// Verifica se a resposta foi bem-sucedida
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("falha ao enviar mensagem, status: %d", resp.StatusCode)
	}

	return nil
}

func GetWebHookInfo() (*WebhookInfo, error) {
	apiURL := fmt.Sprintf(urlTelegram+"/bot%s/getWebhookInfo", botToken)
	slog.Debug(apiURL)

	resp, err := http.Get(apiURL)
	if err != nil {
		slog.Error("Erro ao fazer a requisição:", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Erro ao ler resposta:", err)
		return nil, err
	}

	var info WebhookInfo
	if err := json.Unmarshal(body, &info); err != nil {
		slog.Error("Erro ao decodificar JSON:", err)
		return nil, err
	}

	fmt.Println(info)

	return &info, nil
}

func GetUpdates(offset int) ([]Update, error) {
	url := fmt.Sprintf(urlTelegram+"/bot%s/getUpdates?offset=%d", botToken, offset)
	slog.Debug(url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var updates GetUpdatesResponse
	err = json.Unmarshal(body, &updates)
	if err != nil {
		return nil, err
	}

	if !updates.Ok {
		return nil, fmt.Errorf("erro ao buscar updates")
	}

	return updates.Result, nil
}

func SendMessage(chatID int, message string) error {
	apiURL := fmt.Sprintf(urlTelegram+"/bot%s/sendMessage", botToken)

	// Criando os dados do POST
	data := url.Values{}
	data.Set("chat_id", fmt.Sprint(chatID))
	data.Set("text", message)

	// Fazendo a requisição HTTP
	resp, err := http.PostForm(apiURL, data)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	slog.Info("messeger :: SendMessage :: status code -> " + fmt.Sprint(resp.StatusCode))

	// Verifica se a resposta foi bem-sucedida
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("falha ao enviar mensagem, status: %d", resp.StatusCode)
	}

	return nil
}

func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Erro ao ler corpo da requisição", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var update Update
	if err := json.Unmarshal(body, &update); err != nil {
		http.Error(w, "Erro ao processar JSON", http.StatusInternalServerError)
		return
	}

	fmt.Printf("Mensagem recebida: %s\n", update.Message.Text)

	w.WriteHeader(http.StatusOK)
}
