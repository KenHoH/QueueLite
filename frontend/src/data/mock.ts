import type { Business, Counter, QueueEntry } from "../types";
export const businesses: Business[] = [
  {
    id: "northside",
    name: "Northside Barbers",
    category: "Barbershop",
    location: "Pluit, Jakarta",
    address: "Jl. Pluit Karang Ayu No. 18, North Jakarta",
    description:
      "Good hair. Good company. A neighborhood barbershop for thoughtful cuts, fresh shaves, and a little time for yourself.",
    open: true,
    waiting: 12,
    wait: "30–40",
    counters: 3,
    initials: "NB",
    color: "sage",
    phone: "+62 21 661 2048",
    email: "hello@northside.example",
  },
  {
    id: "greencare",
    name: "GreenCare Clinic",
    category: "Healthcare",
    location: "Muara Karang, Jakarta",
    address: "Jl. Muara Karang Raya No. 42, North Jakarta",
    description:
      "Friendly, attentive care for your everyday health. Visit our neighborhood team for general consultations.",
    open: true,
    waiting: 6,
    wait: "20–30",
    counters: 2,
    initials: "GC",
    color: "blue",
    phone: "+62 21 661 3090",
    email: "hello@greencare.example",
  },
  {
    id: "quickfix",
    name: "QuickFix Service Center",
    category: "Repair & service",
    location: "Pantai Indah Kapuk, Jakarta",
    address: "Jl. Pantai Indah Utara No. 8, North Jakarta",
    description:
      "Dependable help for the devices you use every day. Bring your device in for an assessment with our technicians.",
    open: true,
    waiting: 3,
    wait: "10–15",
    counters: 2,
    initials: "QF",
    color: "sand",
    phone: "+62 21 588 7100",
    email: "hello@quickfix.example",
  },
  {
    id: "studio-nine",
    name: "Studio Nine Salon",
    category: "Beauty & wellness",
    location: "Pluit, Jakarta",
    address: "Jl. Pluit Selatan No. 9, North Jakarta",
    description:
      "A calm space for a fresh look. Expert styling and personal attention, right in your neighborhood.",
    open: false,
    waiting: 0,
    wait: "—",
    counters: 0,
    initials: "SN",
    color: "lilac",
    phone: "+62 21 666 9020",
    email: "hello@studionine.example",
  },
];
export const counters: Counter[] = [
  { id: 1, operator: "Rizky", started: "14:02" },
  { id: 2, operator: "Andi", started: "14:05" },
  { id: 3, operator: "Dimas", started: "14:08" },
];
export const initialEntries: QueueEntry[] = [
  ...[13, 14, 15].map((n, i): QueueEntry => ({
    id: `A-0${n}`,
    businessId: "northside",
    name: ["Daniel", "Kevin", "Budi"][i],
    priority: false,
    status: "Serving",
    service: "General",
    minutes: 0,
    counter: i + 1,
    startedAt: counters[i].started,
  })),
  ...Array.from({ length: 12 }, (_, i): QueueEntry => ({
    id: i === 6 ? "P-003" : `A-0${16 + (i > 6 ? i - 1 : i)}`,
    businessId: "northside",
    name:
      i === 4 ? "Matthew Sutiono" : ["Alex", "William", "Ryan", "Felix"][i % 4],
    priority: i === 6,
    status: "Waiting",
    service: i % 3 === 1 ? "Haircut" : i % 3 === 2 ? "Consultation" : "General",
    minutes: Math.max(1, 18 - i),
    mine: i === 4,
  })),
  ...businesses
    .filter((b) => b.id !== "northside")
    .flatMap((b) =>
      Array.from({ length: b.waiting }, (_, i): QueueEntry => ({
        id: `${b.id === "greencare" ? "G" : "Q"}-${String(i + 1).padStart(3, "0")}`,
        businessId: b.id,
        name: "Guest",
        priority: false,
        status: "Waiting",
        service: "General",
        minutes: 10 - i,
      })),
    ),
];
