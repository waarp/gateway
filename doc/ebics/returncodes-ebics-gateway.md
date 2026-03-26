# Traitement des return codes EBICS dans Gateway

## 1. Objet

Ce document fixe la politique cible de traitement des return codes EBICS dans
Waarp Gateway.

Il sert de prealable a la derivation des routes REST en commandes CLI et a la
definition des comportements d'exploitation.

## 2. Principes structurants

Gateway doit respecter les principes suivants:

- EBICS expose deux scopes distincts de return codes:
  - `technical`
  - `business`
- Gateway ne doit jamais ecraser ces deux scopes dans un champ unique.
- Gateway doit toujours conserver:
  - les codes bruts recus;
  - leur scope;
  - le texte de diagnostic associe quand il existe.
- Gateway peut calculer un statut d'exploitation derive, mais ce statut ne
  remplace pas les return codes EBICS.

## 3. Lecture operationnelle

Ordre de lecture recommande:

1. lire le return code `technical`;
2. lire le return code `business`;
3. deriver un `gatewayOutcome`;
4. deriver une `retryDecision`.

Interpretation de base:

- `technical` de classe `00`, `01` ou `03`:
  - la phase protocolaire est nominale ou avec note/warning;
- `technical` de classe `06`:
  - incident technique potentiellement recuperable;
- `technical` de classe `09`:
  - incident technique ou de securite non recuperable par retry aveugle;
- `technical` nominal + `business` en erreur:
  - le protocole a fonctionne, mais l'ordre est refuse au niveau fonctionnel
    bancaire;
  - il ne faut pas declencher de retry protocolaire automatique.

## 4. Projection dans `EbicsOperation`

`EbicsOperation` doit porter separement:

- `technical_return_code`
- `technical_return_message`
- `business_return_code`
- `business_return_message`
- `gateway_outcome`
- `retry_decision`
- `manual_action_required`

Valeurs cibles de `gateway_outcome`:

- `SUCCESS`
- `SUCCESS_WITH_WARNING`
- `PENDING_BANK`
- `EMPTY_SUCCESS`
- `TECHNICAL_RETRYABLE_FAILURE`
- `TECHNICAL_FATAL_FAILURE`
- `BUSINESS_REJECTED`
- `MANUAL_CONFIRMATION_REQUIRED`

Valeurs cibles de `retry_decision`:

- `NO_RETRY`
- `AUTO_RETRY_ALLOWED`
- `RECOVERY_REQUIRED`
- `MANUAL_REPLAY_ONLY`
- `MANUAL_CONFIRMATION_ONLY`

## 5. Regles de mapping minimales

### 5.1 Cas generiques

- `technical=000000` et `business` absent ou `000000`:
  - `gateway_outcome=SUCCESS`
  - `retry_decision=NO_RETRY`
- `technical` classe `06`:
  - `gateway_outcome=TECHNICAL_RETRYABLE_FAILURE`
  - `retry_decision=AUTO_RETRY_ALLOWED` ou `RECOVERY_REQUIRED` selon le code;
- `technical` classe `09`:
  - `gateway_outcome=TECHNICAL_FATAL_FAILURE`
  - `retry_decision=NO_RETRY` sauf cas explicitement connus de reprise;
- `technical` nominal et `business` classe `06` ou `09`:
  - `gateway_outcome=BUSINESS_REJECTED`
  - `retry_decision=NO_RETRY`
  - `manual_action_required=true`.

### 5.2 Cas EBICS a traiter explicitement

- `EBICS_OK (000000)`:
  - succes nominal;
- `EBICS_TX_RECOVERY_SYNC (061101)`:
  - echec technique recuperable;
  - oriente reprise protocolaire, pas un replay aveugle;
- `EBICS_TX_UNKNOWN_TXID (091101)`:
  - transaction perdue ou inconnue;
  - pas de retry automatique generique;
- `EBICS_TX_MESSAGE_REPLAY (091103)`:
  - anti-rejeu;
  - interdire tout retry automatique aveugle;
- `EBICS_RECOVERY_NOT_SUPPORTED (091105)`:
  - reprise impossible;
  - imposer une decision operateur ou une recreation controlee;
- `EBICS_BANK_PUBKEY_UPDATE_REQUIRED (091008)`:
  - echec technique bloquant;
  - deriver une action de rafraichissement `HPB` / trust material;
- `EBICS_NO_DOWNLOAD_DATA_AVAILABLE (090005)`:
  - resultat operationnel non fatal;
  - ne pas le traiter comme erreur technique generique;
- `EBICS_AMOUNT_CHECK_FAILED (091303)`:
  - rejet bancaire metier;
  - ne pas auto-rejouer;
- `EBICS_ACCOUNT_AUTHORISATION_FAILED (091302)`:
  - rejet bancaire metier;
  - ne pas auto-rejouer;
- `EBICS_DOWNLOAD_POSTPROCESS_DONE (011000)` et
  `EBICS_DOWNLOAD_POSTPROCESS_SKIPPED (011001)`:
  - succes avec information;
- `EBICS_ORDER_PARAMS_IGNORED (031001)`:
  - succes avec warning.

## 6. Regles sur retry, replay et recovery

Gateway doit distinguer strictement:

- `retry`:
  - reenvoi technique controle d'une tentative compatible avec l'idempotence
    et le type d'ordre;
- `replay`:
  - reexecution volontaire d'un ordre ou d'une operation deja emise;
- `recovery`:
  - poursuite de transaction ou de segmentation selon `TxStore`.

Regles:

- un business code ne doit jamais declencher `AUTO_RETRY_ALLOWED`;
- `EBICS_TX_MESSAGE_REPLAY` doit verrouiller tout retry automatique;
- `EBICS_TX_RECOVERY_SYNC` doit orienter vers `RECOVERY_REQUIRED`;
- les ordres sensibles (`INI`, `HIA`, `H3K`, `SPR`, `PUB`, `HCA`, `HCS`,
  `HSA`) doivent rester au minimum en `MANUAL_REPLAY_ONLY` hors cas documente
  de reprise technique.

## 7. Effet sur REST, CLI et UI

REST, CLI et UI doivent toujours exposer:

- `technicalReturnCode`
- `technicalReturnMessage`
- `businessReturnCode`
- `businessReturnMessage`
- `gatewayOutcome`
- `retryDecision`
- `manualActionRequired`

Ils ne doivent pas exposer un unique champ `returnCode` sans precision de
scope.

## 8. Effet sur les statuts d'exploitation

Les statuts d'exploitation Gateway ne doivent pas chercher a reproduire tout le
catalogue EBICS.

Ils doivent:

- rester lisibles pour l'exploitant;
- etre derives des scopes EBICS;
- renvoyer systematiquement vers les codes source.

Exemple:

- `status=FAILED`
- `gatewayOutcome=BUSINESS_REJECTED`
- `technicalReturnCode=000000`
- `businessReturnCode=091303`
