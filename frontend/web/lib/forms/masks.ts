export const russianPhoneMask = "+{7} (000) 000-00-00";
export const russianPhonePlaceholder = "+7 (___) ___-__-__";

const plateLetter = /[АВЕКМНОРСТУХ]/;
const vinCharacter = /[A-HJ-NPR-Z0-9]/;

export const vehiclePlateMask = {
  mask: "L000LL00[0]",
  definitions: { L: plateLetter },
  prepare: uppercase,
};

export const vehicleVinMask = {
  mask: "V".repeat(17),
  definitions: { V: vinCharacter },
  prepare: uppercase,
};

function uppercase(value: string) {
  return value.toUpperCase();
}

export function isVehiclePlate(value: string) {
  return /^[АВЕКМНОРСТУХ][0-9]{3}[АВЕКМНОРСТУХ]{2}[0-9]{2,3}$/.test(value);
}

export function isVehicleVin(value: string) {
  return /^[A-HJ-NPR-Z0-9]{17}$/.test(value);
}

export function preparePhoneInput(value: string, mask: { value: string }) {
  const digits = value.replace(/\D/g, "");

  // Начальная 8 обозначает междугородний префикс. После +7 это уже цифра номера.
  if (value === "8" && (mask.value === "" || mask.value === "+")) return "7";
  if (/^8\d{10}$/.test(digits)) return `7${digits.slice(1)}`;

  return value;
}
