# program to convert language to api request within google calendar
# creds are stored in a .env file 

import speech_recognition as sr
from pocketsphinx import LiveSpeech


speech = LiveSpeech()

for phrase in speech:
    print(phrase)

r = sr.Recognizer()
with sr.Microphone() as source:
    print("hello")
    audio = r.listen(source)
    r.adjust_for_ambient_noise(source)
    # recognize speech using Sphinx
    try:
        print("Sphinx thinks you said " + r.recognize_sphinx(audio))
    except sr.UnknownValueError:
        print("Sphinx could not understand audio")
    except sr.RequestError as e:
        print("Sphinx error; {0}".format(e))
