import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import "./App.css";

const LISTEN_SECONDS = 10;

type ListenState = "idle" | "requesting" | "listening" | "stopping" | "error";

function App() {
  const [listenState, setListenState] = useState<ListenState>("idle");
  const [secondsLeft, setSecondsLeft] = useState<number>(LISTEN_SECONDS);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const audioContextRef = useRef<AudioContext | null>(null);
  const analyserRef = useRef<AnalyserNode | null>(null);
  const dataArrayRef = useRef<Uint8Array<ArrayBuffer> | null>(null);
  const rafRef = useRef<number | null>(null);
  const intervalRef = useRef<number | null>(null);
  const stopTimeoutRef = useRef<number | null>(null);

  const isListening = listenState === "listening";
  const isBusy = listenState === "requesting" || listenState === "stopping";

  const label = useMemo(() => {
    switch (listenState) {
      case "idle":
        return "Tap to listen";
      case "requesting":
        return "Requesting microphone…";
      case "listening":
        return `Listening… ${secondsLeft}s`;
      case "stopping":
        return "Stopping…";
      case "error":
        return "Microphone error";
    }
  }, [listenState, secondsLeft]);

  const stopListening = useCallback(async () => {
    if (listenState === "idle" || listenState === "stopping") return;
    setListenState("stopping");

    if (intervalRef.current) {
      window.clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
    if (stopTimeoutRef.current) {
      window.clearTimeout(stopTimeoutRef.current);
      stopTimeoutRef.current = null;
    }
    if (rafRef.current) {
      window.cancelAnimationFrame(rafRef.current);
      rafRef.current = null;
    }

    analyserRef.current = null;
    dataArrayRef.current = null;

    const stream = streamRef.current;
    streamRef.current = null;
    if (stream) {
      for (const track of stream.getTracks()) track.stop();
    }

    const audioContext = audioContextRef.current;
    audioContextRef.current = null;
    if (audioContext) {
      try {
        await audioContext.close();
      } catch {
        void 0;
      }
    }

    setSecondsLeft(LISTEN_SECONDS);
    setListenState("idle");
  }, [listenState]);

  const startListening = useCallback(async () => {
    if (listenState !== "idle") return;

    setErrorMessage(null);
    setListenState("requesting");

    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          channelCount: 1,
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true,
        },
        video: false,
      });

      streamRef.current = stream;

      const AudioContextImpl =
        window.AudioContext ||
        (window as unknown as { webkitAudioContext: typeof AudioContext })
          .webkitAudioContext;
      const audioContext = new AudioContextImpl();
      audioContextRef.current = audioContext;

      const analyser = audioContext.createAnalyser();
      analyser.fftSize = 2048;
      analyser.smoothingTimeConstant = 0.85;
      analyserRef.current = analyser;

      const source = audioContext.createMediaStreamSource(stream);
      source.connect(analyser);

      dataArrayRef.current = new Uint8Array(
        new ArrayBuffer(analyser.frequencyBinCount)
      );

      setSecondsLeft(LISTEN_SECONDS);
      setListenState("listening");

      intervalRef.current = window.setInterval(() => {
        setSecondsLeft((s) => Math.max(0, s - 1));
      }, 1000);

      stopTimeoutRef.current = window.setTimeout(() => {
        void stopListening();
      }, LISTEN_SECONDS * 1000);

      const draw = () => {
        const canvas = canvasRef.current;
        const analyserNode = analyserRef.current;
        const dataArray = dataArrayRef.current;
        if (!canvas || !analyserNode || !dataArray) return;

        const ctx = canvas.getContext("2d");
        if (!ctx) return;

        const { width, height } = canvas;
        ctx.clearRect(0, 0, width, height);

        analyserNode.getByteFrequencyData(dataArray);

        const barCount = 48;
        const step = Math.max(1, Math.floor(dataArray.length / barCount));
        const centerX = width / 2;
        const centerY = height / 2;
        const radius = Math.min(width, height) * 0.22;
        const maxBar = Math.min(width, height) * 0.22;

        for (let i = 0; i < barCount; i++) {
          const v = dataArray[i * step] / 255;
          const barLen = 6 + v * maxBar;
          const angle = (i / barCount) * Math.PI * 2;
          const x1 = centerX + Math.cos(angle) * radius;
          const y1 = centerY + Math.sin(angle) * radius;
          const x2 = centerX + Math.cos(angle) * (radius + barLen);
          const y2 = centerY + Math.sin(angle) * (radius + barLen);

          const gradient = ctx.createLinearGradient(x1, y1, x2, y2);
          gradient.addColorStop(0, "rgba(124, 92, 255, 0.55)");
          gradient.addColorStop(0.6, "rgba(39, 214, 255, 0.75)");
          gradient.addColorStop(1, "rgba(255, 110, 199, 0.85)");

          ctx.strokeStyle = gradient;
          ctx.lineWidth = 4;
          ctx.lineCap = "round";
          ctx.beginPath();
          ctx.moveTo(x1, y1);
          ctx.lineTo(x2, y2);
          ctx.stroke();
        }

        ctx.fillStyle = "rgba(255,255,255,0.06)";
        ctx.beginPath();
        ctx.arc(centerX, centerY, radius - 10, 0, Math.PI * 2);
        ctx.fill();

        rafRef.current = window.requestAnimationFrame(draw);
      };

      rafRef.current = window.requestAnimationFrame(draw);
    } catch (err) {
      const message =
        err instanceof DOMException && err.name === "NotAllowedError"
          ? "Microphone permission denied. Allow access and try again."
          : err instanceof Error
          ? err.message
          : "Failed to access microphone.";
      setErrorMessage(message);
      setListenState("error");
      await stopListening();
    }
  }, [listenState, stopListening]);

  useEffect(() => {
    const onVisibility = () => {
      if (document.visibilityState !== "visible") {
        void stopListening();
      }
    };
    document.addEventListener("visibilitychange", onVisibility);
    return () => {
      document.removeEventListener("visibilitychange", onVisibility);
      void stopListening();
    };
  }, [stopListening]);

  return (
    <div className="page">
      <div className="shell">
        <header className="top">
          <div className="brand">
            <div className="brandMark" aria-hidden="true" />
            <div className="brandText">
              <div className="brandName">echo-sense</div>
              <div className="brandTag">Song recognition</div>
            </div>
          </div>
        </header>

        <main className="main">
          <section className={`card ${isListening ? "cardListening" : ""}`}>
            <div className="viz">
              <canvas
                ref={canvasRef}
                width={520}
                height={520}
                className="canvas"
              />
              <div
                className={`pulse ${isListening ? "pulseOn" : ""}`}
                aria-hidden="true"
              />
              <div className="centerBadge" aria-hidden="true">
                <svg width="26" height="26" viewBox="0 0 24 24" fill="none">
                  <path
                    d="M12 14a3 3 0 0 0 3-3V6a3 3 0 0 0-6 0v5a3 3 0 0 0 3 3Z"
                    stroke="currentColor"
                    strokeWidth="1.8"
                  />
                  <path
                    d="M19 11a7 7 0 0 1-14 0"
                    stroke="currentColor"
                    strokeWidth="1.8"
                    strokeLinecap="round"
                  />
                  <path
                    d="M12 18v3"
                    stroke="currentColor"
                    strokeWidth="1.8"
                    strokeLinecap="round"
                  />
                </svg>
              </div>
            </div>

            <div className="status">
              <div className="statusTitle">{label}</div>
              <div className="statusSub">
                {errorMessage
                  ? errorMessage
                  : isListening
                  ? "Hold still and let the audio settle."
                  : "Use headphones for best results."}
              </div>
            </div>

            <div className="actions">
              <button
                type="button"
                className={`listenButton ${
                  isListening ? "listenButtonOn" : ""
                }`}
                onClick={() =>
                  isListening ? void stopListening() : void startListening()
                }
                disabled={isBusy}
              >
                <span className="listenIcon" aria-hidden="true">
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
                    <path
                      d="M12 14a3 3 0 0 0 3-3V6a3 3 0 0 0-6 0v5a3 3 0 0 0 3 3Z"
                      stroke="currentColor"
                      strokeWidth="2"
                    />
                    <path
                      d="M19 11a7 7 0 0 1-14 0"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                    />
                    <path
                      d="M12 18v3"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                    />
                  </svg>
                </span>
                <span>{isListening ? "Stop" : "Listen"}</span>
              </button>

              <div className="meta">
                <div className="metaItem">
                  <span className="metaKey">Duration</span>
                  <span className="metaValue">10s</span>
                </div>
                <div className="metaItem">
                  <span className="metaKey">Input</span>
                  <span className="metaValue">Microphone</span>
                </div>
              </div>
            </div>
          </section>
        </main>

        <footer className="footer">
          <span>Built for local recognition experiments.</span>
        </footer>
      </div>
    </div>
  );
}

export default App;
