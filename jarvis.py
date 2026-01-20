import ollama
import sys
from datetime import datetime

try:
    import pyttsx3
    TTS_AVAILABLE = True
except ImportError as e:
    TTS_AVAILABLE = False
    print(f"Warning: pyttsx3 not installed. Error: {e}")
    print(f"Run this command to install for your current Python:")
    print(f"  {sys.executable} -m pip install pyttsx3")
    print()

try:
    import speech_recognition as sr
    STT_AVAILABLE = True
except ImportError as e:
    STT_AVAILABLE = False
    print(f"Warning: speech_recognition not installed. Error: {e}")
    print(f"Run this command to install for your current Python:")
    print(f"  {sys.executable} -m pip install SpeechRecognition pyaudio")
    print()

import threading

def print_separator():
    print("\n" + "-" * 60 + "\n")

def print_assistant_message(message):
    print(f"\033[94mJarvis:\033[0m {message}")

def print_user_message(message):
    print(f"\033[92mSir:\033[0m {message}")

def print_error(message):
    print(f"\033[91mError:\033[0m {message}")

def print_info(message):
    print(f"\033[93m{message}\033[0m")

class TTSEngine:
    def __init__(self):
        if not TTS_AVAILABLE:
            raise Exception("pyttsx3 not available")
        
        self.rate = 175
        self.volume = 0.9
    
    def speak(self, text):
        """Speak text in a separate thread to avoid blocking"""
        def _speak():
            try:
                engine = pyttsx3.init()
                engine.setProperty('rate', self.rate)
                engine.setProperty('volume', self.volume)
                engine.say(text)
                engine.runAndWait()
                engine.stop()
            except Exception as e:
                print_error(f"TTS error: {str(e)}")
        
        thread = threading.Thread(target=_speak)
        thread.daemon = True
        thread.start()
    
    def speak_blocking(self, text):
        """Speak text and wait for completion"""
        try:
            engine = pyttsx3.init()
            engine.setProperty('rate', self.rate)
            engine.setProperty('volume', self.volume)
            engine.say(text)
            engine.runAndWait()
            engine.stop()
        except Exception as e:
            print_error(f"TTS error: {str(e)}")

def main():
    model = "llama2-uncensored:7b" 
    conversation_history = []
    tts_enabled = True
    
    tts = None
    if TTS_AVAILABLE:
        try:
            tts = TTSEngine()
            print_info("Text-to-Speech initialized successfully!")
        except Exception as e:
            print_error(f"Could not initialize TTS: {str(e)}")
            print_info("Continuing without voice output...")
            tts_enabled = False
    else:
        print_info("TTS not available. Install with: pip install pyttsx3")
        tts_enabled = False
    
    conversation_history.append({
        'role': 'system',
        'content': 'You are Jarvis, a helpful AI assistant. You are speaking with your user, whom you address as "Sir". Please share your thoughts on any topic you find interesting however stay on track and answer the users questions directly.'
    })
    
    print("\n" + "=" * 60)
    print("Jarvis v0.8")
    print("=" * 60 + "\n")
    
    if tts and tts_enabled:
        tts.speak("Jarvis online. How may I assist you, Sir?")
    
    try:
        ollama.list()
    except Exception as e:
        print_error("Cannot connect to Ollama. Make sure Ollama is running.")
        print_info("Start Ollama with: ollama serve")
        sys.exit(1)
    
    while True:
        try:
            user_input = input("\033[92mSir:\033[0m ").strip()
            
            if user_input.lower() in ['exit', 'quit']:
                print_info("Goodbye!")
                if tts and tts_enabled:
                    tts.speak_blocking("Goodbye, Sir.")
                break
            
            if user_input.lower() == 'clear':
                conversation_history = [conversation_history[0]] 
                print_info("Conversation history cleared!")
                if tts and tts_enabled:
                    tts.speak("Conversation history cleared.")
                continue
            
            if user_input.lower() == 'save':
                timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
                filename = f"conversation_{timestamp}.txt"
                with open(filename, 'w') as f:
                    for msg in conversation_history:
                        if msg['role'] != 'system': 
                            f.write(f"{msg['role'].upper()}: {msg['content']}\n\n")
                print_info(f"Conversation saved to {filename}")
                if tts and tts_enabled:
                    tts.speak(f"Conversation saved to {filename}")
                continue
            
            if user_input.lower() == 'mute':
                tts_enabled = False
                print_info("Voice output muted.")
                continue
            
            if user_input.lower() == 'unmute':
                if tts:
                    tts_enabled = True
                    print_info("Voice output enabled.")
                    tts.speak("Voice output enabled.")
                else:
                    print_info("TTS not available.")
                continue
            
            if not user_input:
                continue
            
            conversation_history.append({
                'role': 'user',
                'content': user_input
            })
            
            print("\033[94mJarvis:\033[0m ", end="", flush=True)
            
            response_content = ""
            stream = ollama.chat(
                model=model,
                messages=conversation_history,
                stream=True
            )
            
            for chunk in stream:
                content = chunk['message']['content']
                print(content, end='', flush=True)
                response_content += content
            
            print()  
            
            conversation_history.append({
                'role': 'freindly assistant',
                'content': response_content
            })
            
            if tts and tts_enabled:
                tts.speak(response_content)
            
            print_separator()
            
        except KeyboardInterrupt:
            print("\n")
            print_info("Interrupted. Type 'exit' to quit or continue chatting.")
            continue
        except Exception as e:
            print_error(f"An error occurred: {str(e)}")
            import traceback
            traceback.print_exc()
            continue

if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print_error(f"Fatal error: {str(e)}")
        import traceback
        traceback.print_exc()
    finally:
        print("\n" + "=" * 60)
        print("Press Enter to close...")
        print("=" * 60)

        input()
