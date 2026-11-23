"use client";

import { useId, useState, type KeyboardEvent } from "react";
import { Input } from "@/components/ui/input";

export function SuggestionField({ name, label, placeholder, findOptions, noMatches, error, disabled }: {
  name: string;
  label: string;
  placeholder: string;
  findOptions: (input: string) => string[];
  noMatches?: string;
  error?: string;
  disabled?: boolean;
}) {
  const id = useId();
  const [value, setValue] = useState("");
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1);
  const options = findOptions(value);
  const expanded = open && !disabled && options.length > 0;

  function select(option: string) {
    setValue(option);
    setOpen(false);
    setActive(-1);
  }

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === "Escape" && open) {
      event.preventDefault();
      setOpen(false);
      setActive(-1);
    } else if (event.key === "Enter" && expanded && options[active]) {
      event.preventDefault();
      select(options[active]);
    } else if (options.length && (event.key === "ArrowDown" || event.key === "ArrowUp")) {
      event.preventDefault();
      setOpen(true);
      setActive(event.key === "ArrowDown"
        ? Math.min(active + 1, options.length - 1)
        : active <= 0 ? options.length - 1 : active - 1);
    }
  }

  return (
    <div className="space-y-1">
      <label className="text-sm font-medium" htmlFor={id}>{label}</label>
      <Input
        id={id}
        name={name}
        role="combobox"
        autoComplete="off"
        aria-autocomplete="list"
        aria-expanded={expanded}
        aria-controls={expanded ? `${id}-options` : undefined}
        aria-activedescendant={expanded && options[active] ? `${id}-option-${active}` : undefined}
        aria-invalid={Boolean(error)}
        aria-describedby={error ? `${id}-error` : undefined}
        value={value}
        placeholder={placeholder}
        disabled={disabled}
        required
        onFocus={() => setOpen(true)}
        onBlur={() => {
          setOpen(false);
          setActive(-1);
        }}
        onChange={(event) => {
          setValue(event.target.value);
          setActive(-1);
          setOpen(true);
        }}
        onKeyDown={handleKeyDown}
      />
      {expanded && (
        <ul id={`${id}-options`} role="listbox" aria-label={label}
          className="max-h-48 overflow-y-auto rounded-md border bg-card p-1 shadow-sm">
          {options.map((option, index) => (
            <li id={`${id}-option-${index}`} key={option} role="option"
              aria-selected={active === index}
              className="cursor-pointer rounded px-3 py-2 hover:bg-accent aria-selected:bg-accent"
              onMouseDown={(event) => {
                // Фокус остаётся в поле, пока пользователь выбирает вариант мышью.
                event.preventDefault();
              }}
              onClick={() => select(option)}>
              {option}
            </li>
          ))}
        </ul>
      )}
      {open && !disabled && noMatches && options.length === 0 && (
        <p role="status" className="text-sm text-muted-foreground">{noMatches}</p>
      )}
      {error && <p role="alert" id={`${id}-error`} className="text-sm text-destructive">{error}</p>}
    </div>
  );
}
