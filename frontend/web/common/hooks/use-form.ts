"use client";

import { useRef, useState, type FormEvent } from "react";

export type FieldErrors = Record<string, string | undefined>;
export type SubmitForm = (form: FormData) => Promise<void>;

export type FormFeedback = { fields?: FieldErrors; message?: string };

export function useForm({ send, validate, describeError }: {
  send: SubmitForm;
  validate: (form: FormData) => FieldErrors;
  describeError: (error: unknown) => FormFeedback;
}) {
  const submitting = useRef(false);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");
  const [errors, setErrors] = useState<FieldErrors>({});

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting.current) return;

    const data = new FormData(event.currentTarget);
    const invalid = validate(data);
    setMessage("");
    setErrors(invalid);

    if (Object.keys(invalid).length) return;

    submitting.current = true;
    setPending(true);

    void send(data).catch((error: unknown) => {
      const feedback = describeError(error);
      setErrors(feedback.fields ?? {});
      setMessage(feedback.message ?? "");
    }).finally(() => {
      submitting.current = false;
      setPending(false);
    });
  }

  return { pending, message, errors, submit };
}
