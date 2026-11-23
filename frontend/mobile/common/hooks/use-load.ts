import { useCallback, useEffect, useState } from "react";
import { sessionErrorMessage } from "../api/controller";

type LoadState<T> =
  | { status: "loading" }
  | { status: "ready"; data: T }
  | { status: "error"; error: unknown };

export function useLoad<T>(
  load: (signal: AbortSignal) => Promise<T>,
  onSessionEnd: (message: string) => void,
) {
  const [state, setState] = useState<LoadState<T>>({ status: "loading" });
  const [attempt, setAttempt] = useState(0);

  const reload = useCallback(() => {
    setState({ status: "loading" });
    setAttempt((value) => value + 1);
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    setState({ status: "loading" });

    void load(controller.signal).then((data) => {
      if (!controller.signal.aborted) setState({ status: "ready", data });
    }).catch((error: unknown) => {
      if (controller.signal.aborted) return;
      const sessionError = sessionErrorMessage(error);
      if (sessionError) return onSessionEnd(sessionError);
      setState({ status: "error", error });
    });

    return () => controller.abort();
  }, [attempt, load, onSessionEnd]);

  return { state, reload };
}
