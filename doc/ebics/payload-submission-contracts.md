# Contrats de soumission payload EBICS

## 1. Objet

Ce document detaille les contrats de soumission des ordres payload EBICS:

- `BTU`
- `BTD`
- `FUL`
- `FDL`

Il fixe:

- les DTO REST cibles;
- les regles de resolution des parametres;
- les validations minimales;
- la politique produit `profile-required | profile-preferred | free-input-allowed`.

## 2. Principes

- la soumission d'un payload cree une `EbicsOperation`;
- si un flux fichier est materiellement manipule, un `Transfer` est projete;
- le profil payload est le mode nominal recommande;
- les valeurs explicites restent possibles selon la politique de securisation;
- la validation finale se fait toujours contre `ebics_contract_view_items`.

## 3. Politique produit

Trois modes cibles:

- `profile-required`
- `profile-preferred`
- `free-input-allowed`

### 3.1 `profile-required`

Effet:

- un `profile` valide est obligatoire;
- la saisie libre seule est refusee;
- les surcharges explicites restent possibles si elles ne cassent pas la
  validite contractuelle.

Usage cible:

- production fortement gouvernee;
- exploitation bancaire tres encadree.

### 3.2 `profile-preferred`

Effet:

- le `profile` est le mode normal;
- la saisie libre est autorisee mais signalee comme exceptionnelle;
- la CLI et l'API peuvent emettre un warning ou exiger un flag explicite.

Usage cible:

- mode par defaut recommande pour Gateway.

### 3.3 `free-input-allowed`

Effet:

- la saisie libre est autorisee;
- la validation contractuelle reste obligatoire si active;
- convient surtout a l'administration avancee et au bootstrap.

## 4. Resolution des parametres

Ordre de resolution:

1. champs explicites fournis a la commande ou a l'API;
2. champs du `EbicsPayloadProfile`;
3. defaults issus du `ProtoConfig` EBICS;
4. validation contre la vue contractuelle active.

Regles:

- un champ explicite surchage toujours un profil;
- un profil surcharge toujours le `ProtoConfig`;
- aucun niveau de default ne peut contourner un rejet contractuel;
- la resolution doit produire un `resolvedPayloadRequest` exploitable et
  journalisable.

## 5. DTO REST cibles

### 5.1 DTO commun

```go
type InEbicsPayloadRequest struct {
    Profile       string         `json:"profile,omitempty" yaml:"profile,omitempty"`
    Rule          string         `json:"rule,omitempty" yaml:"rule,omitempty"`
    Subscriber    InSubscriberRef `json:"subscriber" yaml:"subscriber"`
    File          *InPayloadFile `json:"file,omitempty" yaml:"file,omitempty"`
    Target        *InPayloadTarget `json:"target,omitempty" yaml:"target,omitempty"`
    Service       *InPayloadService `json:"service,omitempty" yaml:"service,omitempty"`
    Metadata      map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
```

### 5.2 Sous-objets

```go
type InSubscriberRef struct {
    HostID    string `json:"hostId" yaml:"hostId"`
    PartnerID string `json:"partnerId" yaml:"partnerId"`
    UserID    string `json:"userId" yaml:"userId"`
}

type InPayloadFile struct {
    Path       string `json:"path" yaml:"path"`
    OutputName string `json:"outputName,omitempty" yaml:"outputName,omitempty"`
}

type InPayloadTarget struct {
    Directory string `json:"directory,omitempty" yaml:"directory,omitempty"`
}

type InPayloadService struct {
    ServiceName   string `json:"serviceName,omitempty" yaml:"serviceName,omitempty"`
    ServiceOption string `json:"serviceOption,omitempty" yaml:"serviceOption,omitempty"`
    Scope         string `json:"scope,omitempty" yaml:"scope,omitempty"`
    MsgName       string `json:"msgName,omitempty" yaml:"msgName,omitempty"`
    ContainerType string `json:"containerType,omitempty" yaml:"containerType,omitempty"`
}
```

### 5.3 Metadata techniques recommandes

Champs recommandes dans `metadata`:

- `correlationId`
- `declaredAmount`
- `declaredCurrency`
- `requestedDate`
- `fromDate`
- `toDate`
- `businessContext`

## 6. Contraintes par type d'ordre

### 6.1 `BTU`

Minimum:

- `subscriber`
- `file.path`
- `service.serviceName`
- `service.scope`

Souvent attendus:

- `service.serviceOption`
- `service.msgName`
- `metadata.declaredAmount`

### 6.2 `FUL`

Minimum:

- `subscriber`
- `file.path`

Optionnels selon usage:

- informations de service si l'integration choisie les porte.

### 6.3 `BTD`

Minimum:

- `subscriber`
- `target.directory` ou autre strategie de remise;
- `service.serviceName`
- `service.scope`

### 6.4 `FDL`

Minimum:

- `subscriber`
- `target.directory` ou autre strategie de remise.

## 7. Validation

La validation doit suivre cet ordre:

1. valider la forme de la requete;
2. resoudre les champs via `profile` et defaults;
3. verifier la coherence `orderType/direction`;
4. verifier la compatibilite avec `Rule` si une `Rule` est referencee;
5. verifier la compatibilite contractuelle contre
   `ebics_contract_view_items`;
6. verifier les contraintes additionnelles du profil:
   `requiresDeclaredAmount`, extensions autorisees, pattern de nommage.

## 8. Resultat de resolution

Avant emission, Gateway doit pouvoir produire un objet logique du type:

```json
{
  "orderType": "BTU",
  "resolutionMode": "profile-preferred",
  "profile": "sct-corp-credit-transfer",
  "rule": "ebics-send-default",
  "resolvedService": {
    "serviceName": "SCT",
    "serviceOption": "COR",
    "scope": "GLB",
    "msgName": "pain.001"
  },
  "resolvedFile": {
    "path": "/payloads/pain001.xml",
    "outputName": "pain001.xml"
  },
  "resolvedMetadata": {
    "declaredAmount": "1520.45",
    "declaredCurrency": "EUR"
  },
  "contractValidation": {
    "status": "MATCHED",
    "contractViewId": 12,
    "contractItemId": 98
  }
}
```

## 9. Reponses d'erreur cibles

Cas a distinguer:

- `PROFILE_REQUIRED`
- `PROFILE_NOT_FOUND`
- `PROFILE_DISABLED`
- `PROFILE_CONTRACT_MISMATCH`
- `FREE_INPUT_NOT_ALLOWED`
- `CONTRACT_ITEM_NOT_FOUND`
- `DECLARED_AMOUNT_REQUIRED`
- `RULE_INCOMPATIBLE`

## 10. Decision recommandee

La chaine cible est:

- `profile-preferred` par defaut produit;
- `profile-required` pour les environnements tres gouvernes;
- validation contractuelle systematique quand la vue est disponible;
- persistance d'un resultat de resolution exploitable pour audit et support.
