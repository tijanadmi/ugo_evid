export const months = [
  "Januar",
  "Februar",
  "Mart",
  "April",
  "Maj",
  "Jun",
  "Jul",
  "Avgust",
  "Septembar",
  "Oktobar",
  "Novembar",
  "Decembar",
];
export const weekdays = ["Pon", "Uto", "Sre", "Čet", "Pet", "Sub", "Ned"];
export const iso = (year, month, day) =>
  `${year}-${String(month).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
export function localToday() {
  const p = new Intl.DateTimeFormat("en", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    timeZone: "Europe/Belgrade",
  }).formatToParts(new Date());
  return Object.fromEntries(
    p.filter((x) => x.type !== "literal").map((x) => [x.type, Number(x.value)]),
  );
}
