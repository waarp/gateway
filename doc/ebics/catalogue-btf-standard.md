# Catalogue BTF standard

## 1. Objectif

Ce document cadre l'ajout dans Gateway d'un catalogue BTF standard, generique,
non specifique a une banque ou a un partenaire.

Ce catalogue sert a:

- proposer des profils payload pre-valides;
- valider des echanges meme avant recuperation d'un contrat specifique;
- guider l'exploitation et l'administration;
- reduire les erreurs de saisie sur les tuples BTF.

Il ne remplace pas la vue contractuelle specifique publiee par la banque.

## 2. Positionnement

Le catalogue standard a sa place dans Gateway, pas dans `lib-ebics`.

Raison:

- `lib-ebics` doit rester centree sur le protocole et la validation normative;
- le catalogue standard est une aide produit/exploitation;
- son contenu est amene a evoluer dans le temps selon les annexes EBICS;
- il doit pouvoir etre gouverne, versionne et remplace independamment de la lib.

Sources candidates:

- code list embarquee de `lib-ebics`:
  [codelist.go](C:\MonProjet\EBICS\ebics\xsd\codelist.go)
- annexe officielle locale:
  `C:\MonProjet\work-notes-ebics\Spécification_ebics\2024-10-23-EBICS_Annex_BTF-ExternalCodeList.xlsx`

## 3. Frontiere avec les contrats specifiques

Le mecanisme de contrats specifiques est deja en place dans Gateway via:

- `HPD`
- `HKD`
- `HTD`
- `HAA`

Ces ordres alimentent:

- [ebics_contract_view.go](c:\MonProjet\Waarp-Gateway\pkg\model\ebics_contract_view.go)
- [ebics_contract_view_item.go](c:\MonProjet\Waarp-Gateway\pkg\model\ebics_contract_view_item.go)
- [client_contracts.go](c:\MonProjet\Waarp-Gateway\pkg\protocols\modules\ebics\client_contracts.go)

Le catalogue standard ne doit donc pas etre confondu avec:

- un contrat banque;
- une autorisation contractuelle specifique;
- une politique metier client.

## 4. Regle de resolution

Ordre de priorite recommande:

1. vue contractuelle specifique active (`HPD/HKD/HTD/HAA`)
2. catalogue standard pays (`FR`, `DE`, `AT`, `CH`)
3. catalogue standard global (`GLB`)

Regles:

- un contrat specifique actif prime toujours;
- le catalogue pays sert de filet de securite et de baseline de validation;
- `GLB` est la derniere baseline generique;
- aucune entree standard ne doit etre exposee comme "autorisee par la banque"
  tant qu'elle ne vient pas d'un contrat specifique.

## 5. Objet a stocker

Recommendation:

- ne pas reutiliser `ebics_contract_views` pour ce besoin;
- introduire un objet dedie de type `EbicsStandardBTFEntry`
  ou `EbicsStandardProfileTemplate`.

Attributs minimaux:

- `catalog_name`
- `catalog_version`
- `source_type`
- `source_ref`
- `scope`
- `service_name`
- `service_option`
- `msg_name`
- `container_type`
- `direction`
- `order_type`
- `country_group`
- `status`
- `is_default_template`
- `metadata`

Valeurs attendues:

- `scope`: `GLB`, `FR`, `DE`, `AT`, `CH`
- `source_type`: `LIB_EMBEDDED`, `OFFICIAL_ANNEX`, `CUSTOM_OVERRIDE`
- `status`: `ACTIVE`, `SUPERSEDED`, `DISABLED`

## 6. Strategie de persistance

Option recommandee:

- une table dediee du type `ebics_standard_btf_catalog_entries`
- une table d'entete optionnelle `ebics_standard_btf_catalogs`

Pourquoi:

- on separe clairement standard et specifique;
- on garde une gouvernance de version;
- on simplifie REST/UI;
- on permet du seed versionne et du futur override.

## 7. Strategie de seed

La bonne approche est un seed applicatif versionne, pas une logique implicite.

Ordre recommande:

1. charger la base `GLB` depuis la code list embarquee `lib-ebics`
2. charger les catalogues pays `FR`, `DE`, `AT`, `CH`
3. documenter chaque lot avec `catalog_version` et `source_ref`

Le seed doit etre:

- rejouable;
- idempotent;
- traçable;
- non destructif par defaut;
- compatible multi-SGBD via l'ORM existant.

## 8. Lien avec les profils payload

Le catalogue standard doit nourrir `EbicsPayloadProfile`, pas le remplacer.

Usage cible:

- creation assistee de profils a partir d'entrees standard;
- validation des profils contre le catalogue standard si aucun contrat
  specifique n'est encore disponible;
- proposition automatique de profils standards lors de l'onboarding.

## 9. Surfaces d'administration

REST recommande:

- `GET /api/ebics/standard-btf/catalogs`
- `GET /api/ebics/standard-btf/entries`
- `POST /api/ebics/standard-btf/actions/seed`
- `POST /api/ebics/standard-btf/actions/refresh`

CLI recommande:

- `waarp-gateway ebics standard-btf list`
- `waarp-gateway ebics standard-btf show`
- `waarp-gateway ebics standard-btf seed`
- `waarp-gateway ebics standard-btf refresh`

## 10. Regles de validation

Le catalogue standard autorise:

- validation de tuple;
- suggestion de profils;
- garde-fou pre-contractuel.

Il n'autorise pas:

- l'affirmation "ce BTF est contractuellement autorise par la banque";
- le contournement d'un contrat specifique plus restrictif;
- la reconstruction d'une politique metier client.

## 11. Sous-lot recommande

Avant `B4`, ajouter un sous-lot `B3.5 - Catalogue BTF standard`.

Definition of done:

- objet de persistance dedie cadre;
- strategy de seed arretee;
- source `GLB/FR/DE/AT/CH` identifiee;
- ordre de resolution `specific > country > GLB` fige;
- surfaces REST/CLI cadrees;
- difference standard/specifique explicite dans l'admin.
