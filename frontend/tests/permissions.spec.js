import { test, expect } from "@playwright/test";

async function setup(context, options = {}) {
  const state = {
    organizations: [1, 2, 3, 4].map((id) => ({
      id,
      sifra: `OJ${id}`,
      naziv: `Jedinica ${id}`,
      can_write: id === 4,
    })),
    calendar_admin: false,
    ...options,
  };
  const calls = [];
  await context.addInitScript(() =>
    sessionStorage.setItem(
      "ugo-evid-session",
      JSON.stringify({
        access_token: "access",
        refresh_token: "refresh",
        user: { username: "mixed.user" },
        access_token_expires_at: new Date(Date.now() + 3600000).toISOString(),
        refresh_token_expires_at: new Date(Date.now() + 7200000).toISOString(),
      }),
    ),
  );
  await context.route("**/api/**", async (route) => {
    const url = new URL(route.request().url());
    calls.push(url);
    const json = (body, status = 200) => route.fulfill({ json: body, status });
    if (url.pathname.endsWith("/users/prava"))
      return state.fail
        ? json({ error: "Prava nisu dostupna" }, 500)
        : json(state);
    if (url.pathname.endsWith("/podsetnici")) return json({ items: [] });
    if (url.pathname.endsWith("/detalji")) {
      const id = Number(url.pathname.split("/")[3]);
      return json({
        id_ugo_evid: id,
        id_ugo_org: id,
        id_sap_dobavljac: 20,
        br_ugovor: `460${id}`,
        predmet_ugovora: "Održavanje",
        otvoren_ug: "X",
        lica_dobavljaca: [],
      });
    }
    if (url.pathname.endsWith("/dokumenti"))
      return json({
        items: [],
        types: [{ id: 1, sifra: "UGOVOR", naziv: "Ugovor" }],
      });
    if (url.pathname.endsWith("/ugo_tip_ugovora")) return json({ items: [] });
    if (url.pathname.endsWith("/podesavanja"))
      return json({
        settings: { id_tip_ugovora: 0, version: "1", odrzavanje: null },
        document_types: [],
      });
    if (url.pathname.endsWith("/neevidentirani"))
      return json({ items: [], total: 0 });
    if (url.pathname.includes("/radni_kalendar/"))
      return json({ items: [], version: "a".repeat(64) });
    if (url.pathname.endsWith("/excel"))
      return route.fulfill({
        contentType: "application/octet-stream",
        body: "xlsx",
      });
    return json({ items: [], total: 0 });
  });
  return { state, calls };
}

test("mixed roles: only writable OJ is used for registration; list and Excel share OJ filter", async ({
  context,
  page,
}) => {
  const { calls } = await setup(context);
  await page.goto("/ugovori/otvoreni");
  await page
    .getByLabel("Organizaciona jedinica", { exact: true })
    .selectOption("2");
  await expect
    .poll(() =>
      calls.some(
        (u) =>
          u.pathname.endsWith("/otvoreni") &&
          u.searchParams.get("id_ugo_org") === "2",
      ),
    )
    .toBe(true);
  const download = page.waitForEvent("download");
  await page.getByRole("button", { name: "Izvezi u Excel" }).click();
  await download;
  expect(
    calls
      .find((u) => u.pathname.endsWith("/excel"))
      .searchParams.get("id_ugo_org"),
  ).toBe("2");
  await page
    .getByRole("button", { name: "Izaberi ugovor", exact: true })
    .click();
  await expect
    .poll(() =>
      calls.some(
        (u) =>
          u.pathname.endsWith("/neevidentirani") &&
          u.searchParams.get("id_ugo_org") === "4",
      ),
    )
    .toBe(true);
});

test("multiple writable organizations require an explicit selection", async ({
  context,
  page,
}) => {
  const { calls } = await setup(context, {
    organizations: [
      { id: 1, sifra: "OJ1", can_write: true },
      { id: 4, sifra: "OJ4", can_write: true },
    ],
  });
  await page.goto("/ugovori/otvoreni");
  await page
    .getByRole("button", { name: "Izaberi ugovor", exact: true })
    .click();
  const dialog = page.getByRole("dialog", { name: "Izaberi ugovor" });
  await expect(
    dialog.getByText("Izaberite organizacionu jedinicu za unos ugovora."),
  ).toBeVisible();
  expect(calls.some((u) => u.pathname.endsWith("/neevidentirani"))).toBe(false);
  await dialog
    .getByLabel("Organizaciona jedinica", { exact: true })
    .selectOption("4");
  await expect
    .poll(() =>
      calls.some(
        (u) =>
          u.pathname.endsWith("/neevidentirani") &&
          u.searchParams.get("id_ugo_org") === "4",
      ),
    )
    .toBe(true);
});

test("GLE hides contract mutations while IUD enables them only for its OJ", async ({
  context,
  page,
}) => {
  await setup(context);
  await page.goto("/ugovori/detalji/1");
  await expect(page.getByRole("heading", { name: /4601/ })).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Dodaj ugovor", exact: true }),
  ).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Prikaži dokumenta ugovora" }),
  ).toBeVisible();
  await page.goto("/ugovori/detalji/4");
  await expect(
    page.getByRole("button", { name: "Dodaj ugovor", exact: true }),
  ).toBeVisible();
});

test("read-only user and failed rights lookup cannot register contracts", async ({
  context,
  page,
}) => {
  const { state } = await setup(context, {
    organizations: [{ id: 1, sifra: "OJ1", can_write: false }],
  });
  await page.goto("/ugovori/otvoreni");
  await expect(
    page.getByRole("heading", { name: "Otvoreni ugovori", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Izaberi ugovor", exact: true }),
  ).toHaveCount(0);
  state.fail = true;
  await page.reload();
  await expect(page.getByRole("region", { name: "Otvoreni ugovori" }).getByRole("alert")).toContainText("Prava nisu dostupna");
  await expect(
    page.getByRole("button", { name: "Izaberi ugovor", exact: true }),
  ).toHaveCount(0);
});

test("calendar editing is a separate permission, including an administrator without any OJ", async ({
  context,
  page,
}) => {
  const { state } = await setup(context);
  await page.goto("/ugovori/radni-kalendar");
  await expect(
    page.getByRole("button", { name: "Generiši kalendar" }),
  ).toBeDisabled();
  state.calendar_admin = true;
  state.organizations = [];
  await page.reload();
  await expect(
    page.getByRole("button", { name: "Generiši kalendar" }),
  ).toBeEnabled();
});
