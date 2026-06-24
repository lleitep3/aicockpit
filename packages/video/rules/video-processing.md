# Otimização de Custo em Análise de Vídeo

Esta regra define o protocolo obrigatório para processamento e análise de arquivos de vídeo, visando a otimização extrema de consumo de tokens.

## Diretriz Principal

Sempre que o desenvolvedor solicitar a análise, descrição ou depuração baseada em um arquivo de vídeo (ex: `.mp4`, `.avi`, `.mov`, `.mkv`), você **NÃO DEVE** analisar o arquivo de vídeo diretamente se puder fatiá-lo. In vez disso, você **DEVE fatiar o vídeo em frames individuais de imagem** e realizar a análise sobre essas imagens.

## Por que fatiar?
* Enviar vídeo diretamente força a API do Gemini a amostrar a 1 frame por segundo por padrão (cada frame custa 258 tokens). Um vídeo de 1 minuto consumirá cerca de 15.480 tokens de imagem, além de tokens de áudio.
* Fatiar o vídeo a cada 5 segundos reduz as imagens necessárias para apenas 12 frames, reduzindo o custo em 5 vezes (~3.096 tokens).

## Protocolo de Ação
1. **Identificar o Vídeo:** Localize o caminho do arquivo de vídeo no prompt do usuário ou no workspace.
2. **Executar o Fatiamento:** Rode o comando `cockpit video slice <path/do/video>` no terminal.
   * Se precisar de maior resolução temporal para movimentos rápidos, passe `--interval 2` ou `--interval 3`.
   * Por padrão, o comando extrairá um frame a cada 5 segundos e salvará em `~/.cockpit/workspace/video-slice/<video-name>/slices`.
3. **Análise de Imagens:** Leia as imagens geradas na pasta informada pelo comando utilizando a ferramenta de visualização de arquivos do agente (`view_file`).
4. **Exceção (Vídeo Direto):** Só envie o vídeo diretamente se a análise exigir explicitamente a trilha de áudio ou movimentos milissegundo a milissegundo que se perderiam na amostragem estática.
