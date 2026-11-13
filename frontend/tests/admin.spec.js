import { test, expect } from "@playwright/test";

async function setup(context, options = {}) {
  const state = {
    admin: true,
    conflict: false,
    lastAdmin: false,
    writes: [],
    users: [
      {
        id: 2,
        ad_sifra: "test.user",
        ime: "Test korisnik",
        status: "A",
        version: "1",
        roles: [{ id_ugo_rola: 1, opseg: "OJ", id_ugo_org: 1 }],
      },
    ],
    notification: {
      id: 8,
      vrsta: "DOC_DELETE",
      id_sloga: 11,
      naslov: "Brisanje dokumenta zahteva proveru",
      poruka: "Permission denied",
      status: "OTVORENO",
      version: "1",
      originalni_naziv: "ugovor.pdf",
      id_ugo_evid: 9,
      broj_pokusaja: 16,
      broj_ponavljanja: 1,
      prvi_put: "2026-11-12T10:00:00",
      poslednji_put: "2026-11-12T10:15:00",
    },
    ...options,
  };
  const calls = [];
  await context.addInitScript(() =>
    sessionStorage.setItem(
      "ugo-evid-session",
      JSON.stringify({
        user: { username: "admin" },
        access_token: "access",
        refresh_token: "refresh",
        access_token_expires_at: new Date(Date.now() + 3600000).toISOString(),
        refresh_token_expires_at: new Date(Date.now() + 7200000).toISOString(),
      }),
    ),
  );
  await context.route("**/api/**", async (route) => {
    const request = route.request(),
      url = new URL(request.url()),
      path = url.pathname;
    calls.push(path);
    const json = (body, status = 200) => route.fulfill({ json: body, status });
    if (path === "/api/users/prava")
      return json({
        admin: state.admin,
        calendar_admin: true,
        organizations: [],
      });
    if (!state.admin)
      return json({ error: "Nemate administratorska prava." }, 403);
    if (path === "/api/admin/sifarnici")
      return json({
        roles: [
          { id: 1, sifra: "GLE", naziv: "Pregled" },
          { id: 2, sifra: "IUD", naziv: "Izmene" },
          { id: 3, sifra: "ADM_KALENDAR", naziv: "Kalendar" },
          { id: 4, sifra: "ADMIN", naziv: "Administrator" },
        ],
        organizations: [
          { id: 1, sifra: "OJ1", naziv: "Prva" },
          { id: 4, sifra: "OJ4", naziv: "Četvrta" },
        ],
      });
    if (path === "/api/admin/korisnici" && request.method() === "GET") {
      const filter = url.searchParams.get("filter") || "";
      const users = state.users.filter((u) =>
        `${u.ad_sifra} ${u.ime}`.includes(filter),
      );
      return json({ items: users, total: users.length });
    }
    if (path.startsWith("/api/admin/korisnici/") && request.method() === "GET")
      return json(
        state.users.find((u) => u.id === Number(path.split("/").at(-1))),
      );
    if (path.startsWith("/api/admin/korisnici") && request.method() !== "GET") {
      const body = request.postDataJSON();
      state.writes.push(body);
      if (state.lastAdmin)
        return json(
          {
            error:
              "Nije dozvoljeno uklanjanje poslednjeg aktivnog administratora.",
          },
          409,
        );
      if (state.conflict)
        return json(
          { error: "Podaci su promenjeni. Ponovo učitajte aktuelne podatke." },
          409,
        );
      const id =
        request.method() === "POST" ? 3 : Number(path.split("/").at(-1));
      const saved = { ...body, id, version: "2" };
      state.users = [...state.users.filter((u) => u.id !== id), saved];
      return json(saved, request.method() === "POST" ? 201 : 200);
    }
    if (path === "/api/admin/obavestenja") {
      const status = url.searchParams.get("status") || "";
      const items =
        status === "SVE" ||
        status === state.notification.status ||
        (!status && state.notification.status !== "RESENO")
          ? [state.notification]
          : [];
      return json({ items, total: items.length });
    }
    if (path === "/api/admin/obavestenja/8") {
      const body = request.postDataJSON();
      state.writes.push(body);
      state.notification = {
        ...state.notification,
        status: body.action === "PREUZMI" ? "PREUZETO" : "OTVORENO",
        preuzeo: body.action === "PREUZMI" ? "admin" : "",
        version: String(Number(state.notification.version) + 1),
      };
      return json({ status: "ok" });
    }
    if (path === "/api/admin/istorija")
      return json({
        items: [
          {
            id: 1,
            korisnik: "admin",
            akcija: "IZMENA_KORISNIKA",
            vrsta: "KORISNIK",
            id_sloga: 2,
            vreme: "2026-11-12T10:20:00",
            pre: '{"status":"A"}',
            posle: '{"status":"N"}',
          },
        ],
        total: 1,
      });
    return json({ items: [], total: 0 });
  });
  return { state, calls };
}

test("administrator can add IUD to their own account while retaining ADMIN", async ({
  context,
  page,
}) => {
  await page.setViewportSize({ width: 1366, height: 768 });
  const { state } = await setup(context, {
    users: [
      {
        id: 1,
        ad_sifra: "admin",
        ime: "Administrator",
        status: "A",
        version: "1",
        roles: [
          { id_ugo_rola: 4, opseg: "GLOBALNO", id_ugo_org: 0 },
          { id_ugo_rola: 3, opseg: "GLOBALNO", id_ugo_org: 0 },
          { id_ugo_rola: 1, opseg: "OJ", id_ugo_org: 1 },
        ],
      },
    ],
  });
  await page.goto("/administracija/korisnici");
  await page
    .getByRole("button", { name: "Korisnik i prava", exact: true })
    .click();
  const dialog = page.getByRole("dialog", { name: "Korisnik i prava" });
  const add = dialog.getByRole("button", { name: "Dodaj pravo", exact: true });
  await expect(add).toBeEnabled();
  for (const viewport of [
    { width: 1366, height: 768 },
    { width: 900, height: 560 },
    { width: 390, height: 844 },
  ]) {
    await page.setViewportSize(viewport);
    await add.click({ trial: true });
  }
  await add.click();
  await dialog
    .getByRole("combobox", { name: "Rola 4", exact: true })
    .selectOption("2");
  await dialog
    .getByRole("combobox", { name: "OJ 4", exact: true })
    .selectOption("4");
  await dialog.getByRole("button", { name: "Sačuvaj korisnika" }).click();
  await expect(dialog).toHaveCount(0);
  expect(state.writes[0].roles).toEqual([
    { id_ugo_rola: 4, opseg: "GLOBALNO", id_ugo_org: 0 },
    { id_ugo_rola: 3, opseg: "GLOBALNO", id_ugo_org: 0 },
    { id_ugo_rola: 1, opseg: "OJ", id_ugo_org: 1 },
    { id_ugo_rola: 2, opseg: "OJ", id_ugo_org: 4 },
  ]);
});

test("calendar admin cannot open global administration", async ({
  context,
  page,
}) => {
  const { calls } = await setup(context, { admin: false });
  await page.goto("/administracija/korisnici");
  await expect(page.getByRole("alert")).toContainText(
    "Nemate administratorska prava",
  );
  expect(calls.some((p) => p.startsWith("/api/admin/"))).toBe(false);
  await expect(page.getByRole("button", { name: "Novi korisnik" })).toHaveCount(
    0,
  );
});

test("ADMIN without OJ creates an account with explicit IUD scope", async ({
  context,
  page,
}) => {
  const { state } = await setup(context);
  await page.goto("/administracija/korisnici");
  await page.getByRole("button", { name: "Novi korisnik" }).click();
  const dialog = page.getByRole("dialog", { name: "Novi korisnik" });
  await dialog.getByLabel("AD korisničko ime").fill("new.user");
  await dialog.getByLabel("Ime i prezime").fill("Novi korisnik");
  await dialog.getByRole("button", { name: "Dodaj pravo" }).click();
  await dialog
    .getByRole("combobox", { name: "Rola 1", exact: true })
    .selectOption("2");
  await dialog
    .getByRole("combobox", { name: "OJ 1", exact: true })
    .selectOption("4");
  await dialog.getByText("Efektivna prava po trenutnom izboru").click();
  await expect(dialog).toContainText(
    "OJ4 — Četvrta: pregled, unos, izmena i brisanje",
  );
  await dialog.getByRole("button", { name: "Sačuvaj korisnika" }).click();
  await expect(dialog).toHaveCount(0);
  expect(state.writes[0].roles).toEqual([
    { id_ugo_rola: 2, opseg: "OJ", id_ugo_org: 4 },
  ]);
  await expect(
    page.getByRole("cell", { name: "new.user", exact: true }),
  ).toBeVisible();
});

test("editing supports all-OJ and global roles and preserves draft after a version conflict", async ({
  context,
  page,
}) => {
  const { state } = await setup(context, { conflict: true });
  await page.goto("/administracija/korisnici");
  await page
    .getByRole("button", { name: "Korisnik i prava", exact: true })
    .click();
  const dialog = page.getByRole("dialog", { name: "Korisnik i prava" });
  await expect(dialog.getByLabel("AD korisničko ime")).toBeDisabled();
  await dialog.getByLabel("Ime i prezime").fill("Sačuvaj ovaj unos");
  await dialog.getByLabel("Opseg 1").selectOption("SVE_OJ");
  await dialog.getByRole("button", { name: "Dodaj pravo" }).click();
  await dialog.getByLabel("Rola 2").selectOption("4");
  await expect(dialog.getByLabel("Opseg 2")).toHaveValue("GLOBALNO");
  await dialog.getByRole("button", { name: "Sačuvaj korisnika" }).click();
  await expect(dialog.getByLabel("Ime i prezime")).toHaveValue(
    "Sačuvaj ovaj unos",
  );
  await expect(
    dialog.getByRole("button", { name: "Sačuvaj korisnika" }),
  ).toBeDisabled();
  await dialog.getByRole("button", { name: "Učitaj aktuelne podatke" }).click();
  await expect(dialog.getByLabel("Ime i prezime")).toHaveValue("Test korisnik");
});

test("last-admin rejection remains visible and does not close the editor", async ({
  context,
  page,
}) => {
  await setup(context, { lastAdmin: true });
  await page.goto("/administracija/korisnici");
  await page
    .getByRole("button", { name: "Korisnik i prava", exact: true })
    .click();
  const dialog = page.getByRole("dialog", { name: "Korisnik i prava" });
  await dialog.getByLabel("Status korisnika").selectOption("N");
  await dialog.getByRole("button", { name: "Sačuvaj korisnika" }).click();
  await expect(
    page.getByText(
      "Nije dozvoljeno uklanjanje poslednjeg aktivnog administratora.",
      { exact: true },
    ),
  ).toBeVisible();
  await expect(dialog).toBeVisible();
});

test("notification badge, claim and automatic resolution share a single incident", async ({
  context,
  page,
}) => {
  const { state } = await setup(context);
  await page.goto("/administracija/obavestenja");
  await expect(
    page.getByRole("link", { name: "Administracija (1)" }),
  ).toBeVisible();
  await expect(page.getByText("ugovor.pdf", { exact: true })).toBeVisible();
  await page.getByText("Detalji greške").click();
  await expect(
    page.getByText("Permission denied", { exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Preuzmi problem" }).click();
  await expect(
    page.getByRole("button", { name: "Vrati u otvorene" }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Označi kao rešeno" }),
  ).toHaveCount(0);
  state.notification.status = "RESENO";
  state.notification.reseno = "2026-11-12T10:30:00";
  await page.getByRole("button", { name: "Osveži", exact: true }).click();
  await expect(
    page.getByText("Nema obaveštenja za izabrani status."),
  ).toBeVisible();
  await page.getByLabel("Status obaveštenja").selectOption("RESENO");
  await expect(
    page.getByText("12.11.2026 10:30:00", { exact: true }),
  ).toBeVisible();
});

test("audit history displays who changed the data and before/after values", async ({
  context,
  page,
}) => {
  await setup(context);
  await page.goto("/administracija/istorija");
  await expect(
    page.getByRole("cell", { name: "12.11.2026 10:20:00", exact: true }),
  ).toBeVisible();
  await page.getByText("Pre i posle", { exact: true }).click();
  await expect(page.locator("pre").first()).toContainText('"A"');
  await expect(page.locator("pre").last()).toContainText('"N"');
});
