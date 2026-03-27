# Design technique - Catalogue BTF standard

## 1. Objet

Ce document derive le sous-lot `B3.5` en design technique concret.

Il couvre:

- le modele SQL cible;
- les structs `model` cibles;
- la strategie de seed versionne;
- le fallback runtime dans la validation contractuelle;
- les DTO REST/CLI minimaux.

## 2. Principe de resolution

Ordre de resolution retenu:

1. contrat specifique actif (`EbicsContractView` + `EbicsContractViewItem`)
2. catalogue standard pays (`FR`, `DE`, `AT`, `CH`)
3. catalogue standard global (`GLB`)

Effets:

- un contrat specifique actif garde la priorite absolue;
- le catalogue standard ne sert qu'en fallback;
- l'origine de la validation doit rester visible.

## 3. Modele SQL cible

## 3.1 `ebics_standard_btf_catalogs`

Role:

- entete versionne d'un catalogue standard.

Champs proposes:

- `id`
- `owner`
- `name`
- `scope`
- `catalog_version`
- `source_type`
- `source_ref`
- `status`
- `seed_checksum`
- `created_at`
- `updated_at`

Contraintes:

- unicite `(owner, name, scope, catalog_version)`
- `scope` dans `GLB`, `FR`, `DE`, `AT`, `CH`
- `status` dans `ACTIVE`, `SUPERSEDED`, `DISABLED`

## 3.2 `ebics_standard_btf_entries`

Role:

- lignes effectives de tuples BTF standards.

Champs proposes:

- `id`
- `owner`
- `catalog_id`
- `entry_key`
- `order_type`
- `direction`
- `service_name`
- `service_option`
- `scope`
- `msg_name`
- `container_type`
- `country_group`
- `is_default_template`
- `status`
- `metadata`
- `created_at`
- `updated_at`

Contraintes:

- unicite `(owner, catalog_id, entry_key)`
- `order_type` borne a `BTU` / `BTD`
- `direction` borne a `UPLOAD` / `DOWNLOAD`
- `status` borne a `ACTIVE` / `DISABLED`
- `scope` porte le `service_scope` du tuple BTF et peut differer du `scope`
  du catalogue; le `scope` du catalogue identifie le panier de fallback
  (`GLB` ou pack pays), pas le `service_scope` impose a chaque ligne

## 4. Structs `model`

## 4.1 `model.EbicsStandardBTFCatalog`

```go
type EbicsStandardBTFCatalog struct {
    ID             int64     `xorm:"<- id AUTOINCR"`
    Owner          string    `xorm:"owner"`
    Name           string    `xorm:"name"`
    Scope          string    `xorm:"scope"`
    CatalogVersion string    `xorm:"catalog_version"`
    SourceType     string    `xorm:"source_type"`
    SourceRef      string    `xorm:"source_ref"`
    Status         string    `xorm:"status"`
    SeedChecksum   string    `xorm:"seed_checksum"`
    CreatedAt      time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
    UpdatedAt      time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}
```

## 4.2 `model.EbicsStandardBTFEntry`

```go
type EbicsStandardBTFEntry struct {
    ID                int64     `xorm:"<- id AUTOINCR"`
    Owner             string    `xorm:"owner"`
    CatalogID         int64     `xorm:"catalog_id"`
    EntryKey          string    `xorm:"entry_key"`
    OrderType         string    `xorm:"order_type"`
    Direction         string    `xorm:"direction"`
    ServiceName       string    `xorm:"service_name"`
    ServiceOption     string    `xorm:"service_option"`
    Scope             string    `xorm:"scope"`
    MsgName           string    `xorm:"msg_name"`
    ContainerType     string    `xorm:"container_type"`
    CountryGroup      string    `xorm:"country_group"`
    IsDefaultTemplate bool      `xorm:"is_default_template"`
    Status            string    `xorm:"status"`
    Metadata          string    `xorm:"metadata"`
    CreatedAt         time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
    UpdatedAt         time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}
```

## 5. Strategie de seed

Le seed doit etre applicatif, versionne et idempotent.

Source initiale recommandee:

- `GLB`: derivee de `lib-ebics` et de sa code list embarquee
- `FR`, `DE`, `AT`, `CH`: derivees de l'annexe officielle locale
- les packs curated `CSV` peuvent contenir, dans un meme catalogue pays,
  des lignes de `service_scope` `GLB` et des lignes de `service_scope` pays

Regles:

- un seed identifie un `catalog_version`;
- l'execution ne supprime pas automatiquement un catalogue actif;
- un nouveau seed active la nouvelle version et supersede l'ancienne.

## 6. Fallback runtime

Le runtime de validation contractuelle doit evoluer.

Nouveau principe:

- `ContractViewResolver` devient `ValidationCatalogResolver`
- il doit savoir resoudre:
  - contrat specifique actif
  - catalogue standard pays
  - catalogue standard `GLB`

Resultat cible:

```go
type ContractValidationResult struct {
    Status            string
    Message           string
    ValidationSource  string
    ContractViewID    int64
    StandardCatalogID int64
    MatchedItems      []model.EbicsContractViewItem
    MatchedTemplates  []model.EbicsStandardBTFEntry
}
```

Valeurs recommandees pour `ValidationSource`:

- `SPECIFIC_CONTRACT`
- `STANDARD_COUNTRY_CATALOG`
- `STANDARD_GLB_CATALOG`
- `NONE`

## 7. Adaptation du runtime

Fichiers cibles:

- `pkg/protocols/modules/ebics/runtime/contract_validation.go`
- nouveau resolver de catalogue standard
- adaptation des resolvers REST/client/server existants

Comportement:

- si un contrat specifique actif existe, il devient exclusif pour l'autorisation;
- si le tuple demande n'est pas trouve dans ce contrat specifique actif,
  la requete doit etre rejetee comme non autorisee par le contrat;
- dans ce cas, aucun fallback vers le catalogue standard ne doit etre tente,
  meme si le tuple existe dans `FR`, `DE`, `AT`, `CH` ou `GLB`;
- le catalogue standard n'est utilise qu'en absence de contrat specifique actif;
- sinon on tente le catalogue du scope demande;
- sinon on tente `GLB`;
- si aucun match, rejet propre avec source explicite.

Message d'erreur cible:

- "the resolved payload request is not authorized by the active EBICS contract"

## 8. DTO REST minimaux

Catalogues:

- `OutEbicsStandardBTFCatalog`
- `OutEbicsStandardBTFEntry`

Actions:

- `InEbicsStandardBTFSeed`
- `OutEbicsStandardBTFSeedResult`

Exposition minimale:

- `GET /api/ebics/standard-btf/catalogs`
- `GET /api/ebics/standard-btf/catalogs/{catalog}`
- `GET /api/ebics/standard-btf/entries`
- `POST /api/ebics/standard-btf/actions/seed`

Fichiers d'exemple importables:

- `pkg/backup/testdata/ebics-standard-btf-curated.json`
- `pkg/backup/testdata/ebics-standard-btf-curated.yaml`

## 9. CLI minimale

- `waarp-gateway ebics standard-btf catalogs`
- `waarp-gateway ebics standard-btf entries`
- `waarp-gateway ebics standard-btf seed`

## 10. Impact sur `EbicsPayloadProfile`

Pas de duplication lourde.

Le profil peut garder:

- `service_name`
- `service_option`
- `scope`
- `msg_name`
- `container_type`

Mais il peut en plus referencer optionnellement:

- `standard_btf_entry_id`

Ce lien n'est pas obligatoire, mais il simplifie:

- la creation assistee;
- la traçabilite;
- la revalidation lors d'un changement de catalogue.

## 11. Definition of done

- tables ciblees cadrees;
- structs `model` cadrees;
- strategie de seed arretee;
- fallback runtime defini;
- source de validation explicite en REST/CLI;
- separation standard/specifique preservee.
