# P.A.D.R (Programmatic AI Daily Reporter)
Some of the things I find annoying are managing my calendar, reading emails, and deciding what to wear based on the weather. So, over Thanksgiving break, I decided to create this automated, AI-powered program
that reads responses from APIs, feeds them into Ollama, and then sends that Ollama-generated response to my phone using AWS SNS. 

### Getting started
To set this up for yourself, you need:
1. To create an OAuth 2.0 client on Gmail (or your preferred mail provider) and enable API usage. [Using OAuth 2.0 to Access Google APIs Authorization](https://developers.google.com/identity/protocols/oauth2)
2. Google has also provided code that you can use to set up all the interactions with their API that I copied into my program. [Go quickstart](https://developers.google.com/workspace/gmail/api/quickstart/go)
3. Set up API access via OpenWeather and save the key to an env variable called **WEATHER_API_KEY**. [OpenWeather API page](https://openweathermap.org/api)
4. Install Ollama locally [Ollama install page](https://ollama.com/download)
5. Make sure to download a model. I used Ollama3.1 for this project, but mostly all of them are fine, some are a bit better at certain things (CV and whatnot), but I picked a general-purpose model since it would be fast and didn't &nbsp
need to do much except summarize some information. [Ollama models](https://ollama.com/search)
---

# To run the project
1. To run this project, ensure you have everything installed and a module created.
  - ```go mod init <module_name>```
3. Then run this command to install everything necessary for the module
  - ```go mod tidy```
3. Start the ollama service
  - ```ollama serve```
4. run the program
  - ```go run main.go```
