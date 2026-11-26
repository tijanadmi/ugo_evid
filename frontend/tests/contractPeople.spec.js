import { test, expect } from "@playwright/test";

async function setup(context, writable = true) {
  const state = {
    writes: [],
    items: [
      {
        id: 10,
        id_ugo_dob_lica: 7,
        id_ugo_dob_lica_rola: 1,
        ime: "Ana",
        rola: "Service Level Manager",
        status: "A",
        status_lica: "A",
        version: "1",
      },
    ],
    conflict: false,
  };
  await context.addInitScript(() =>
    sessionStorage.setItem(
      "ugo-evid-session",
      JSON.stringify({
        access_token: "access",
        refresh_token: "refresh",
        access_token_expires_at: new Date(Date.now() + 3600000).toISOString(),
        refresh_token_expires_at: new Date(Date.now() + 7200000).toISOString(),
        user: { username: "test.user" },
      }),
    ),
  );
  await context.route("**/api/**", async (route) => {
    const request = route.request(),
      path = new URL(request.url()).pathname;
    const json = (data, status = 200) => route.fulfill({ json: data, status });
    if (path.endsWith("/users/prava"))
      return json({
        organizations: [
          { id: 3, sifra: "OJ", naziv: "Moja OJ", can_write: writable },
        ],
      });
    if (path.endsWith("/detalji"))
      return json({
        id_ugo_evid: 9,
        id_ugo_org: 3,
        id_sap_dobavljac: 20,
        br_ugovor: "460",
        predmet_ugovora: "Podrška",
        naziv: "Partner",
        otvoren_ug: "X",
        lica_dobavljaca: [],
      });
    if (path.endsWith("/lica")) {
      if (request.method() !== "GET") {
        const body = request.postDataJSON();
        state.writes.push(body);
        if (state.conflict)
          return json({ error: "Angažovanja su promenjena." }, 409);
        if (request.method() === "DELETE")
          state.items.find((p) => p.id === body.id).status = "P";
        else {
          if (body.id_ugo_dob_lica_rola === 1)
            state.items
              .filter((p) => p.id_ugo_dob_lica_rola === 1)
              .forEach((p) => (p.status = "P"));
          state.items.push({
            ...body,
            id: 11,
            ime: "Bojan",
            rola: "Service Level Manager",
            status: "A",
            status_lica: "A",
            version: "1",
          });
        }
        return json({ status: "ok" });
      }
      return json({
        items: state.items,
        candidates: [
          { id_ugo_dob_lica: 7, ime: "Ana" },
          { id_ugo_dob_lica: 8, ime: "Bojan" },
        ],
      });
    }
    if (path.endsWith("/ugo_dob_lica_rola"))
      return json({
        items: [
          { id: 1, naziv: "Service Level Manager" },
          { id: 2, naziv: "Tehnička podrška" },
        ],
        total: 2,
      });
    if (path.endsWith("/podesavanja"))
      return json({ settings: { version: "1" }, document_types: [] });
    if (path.endsWith("/ugo_tip_ugovora")) return json({ items: [] });
    if (path.endsWith("/dokumenti")) return json({ items: [], types: [] });
    return json({ items: [] });
  });
  return state;
}

test("replace SLM only on this contract and preserve previous assignment", async ({
  page,
  context,
}) => {
  const state = await setup(context);
  await page.goto("/ugovori/detalji/9");
  await page.getByRole("button", { name: "Promeni SLM-a" }).click();
  const dialog = page.getByRole("dialog", { name: "Lice na ugovoru" });
  await dialog.getByLabel("Lice dobavljača").selectOption("8");
  await dialog.getByRole("button", { name: "Sačuvaj angažovanje" }).click();
  await expect(dialog).toHaveCount(0);
  await expect(page.getByRole("list", { name: "Lica ugovora" })).toContainText(
    "Bojan",
  );
  await expect(
    page.getByRole("list", { name: "Lica ugovora" }),
  ).not.toContainText("Ana");
  expect(state.writes[0]).toEqual({
    id_ugo_dob_lica: 8,
    id_ugo_dob_lica_rola: 1,
    prethodni_slm: 10,
    prethodna_verzija: "1",
  });
  await page.getByText("Prethodna angažovanja").click();
  await expect(page.locator("details")).toContainText("Ana");
});

test("stale replacement stays open and viewer has no mutation actions", async ({
  page,
  context,
}) => {
  const state = await setup(context);
  state.conflict = true;
  await page.goto("/ugovori/detalji/9");
  await page.getByRole("button", { name: "Promeni SLM-a" }).click();
  const dialog = page.getByRole("dialog", { name: "Lice na ugovoru" });
  await dialog.getByLabel("Lice dobavljača").selectOption("8");
  await dialog.getByRole("button", { name: "Sačuvaj angažovanje" }).click();
  await expect(
    dialog.getByRole("button", { name: "Sačuvaj angažovanje" }),
  ).toBeDisabled();
  await dialog.getByRole("button", { name: "Odustani" }).click();
  await expect(page.getByRole("list", { name: "Lica ugovora" })).toContainText(
    "Ana",
  );
});

test("read-only organization can view assigned people but cannot change them", async ({
  page,
  context,
}) => {
  await setup(context, false);
  await page.goto("/ugovori/detalji/9");
  await expect(page.getByRole("list", { name: "Lica ugovora" })).toContainText(
    "Ana",
  );
  await expect(page.getByRole("button", { name: "Promeni SLM-a" })).toHaveCount(
    0,
  );
  await expect(
    page.getByRole("button", { name: "Dodaj lice na ugovor" }),
  ).toHaveCount(0);
});
