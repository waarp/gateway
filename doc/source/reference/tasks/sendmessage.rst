.. _ref-task-sendmessage:

SENDMESSAGE
===========


Le traitement ``SENDMESSAGE`` envoie un acquittement PeSIT (F.MESSAGE) à un
partenaire distant. Il est utilisé pour les acquittements applicatifs de bout
en bout dans les architectures *Store and Forward*.

Cette tâche est **spécifique au protocole PeSIT**. Si elle est configurée sur
une règle utilisée par un autre protocole, elle est silencieusement ignorée
(pas d'erreur de transfert).

Fonctionnement
--------------

La tâche ouvre une connexion PeSIT dédiée vers le partenaire, envoie un
F.MESSAGE contenant l'identifiant du transfert et un message texte, puis
ferme la connexion.

**Tous les paramètres sont optionnels.** Le partenaire et le compte sont
résolus automatiquement lorsque le mécanisme de routage Store & Forward est
configuré (voir ci-dessous).

**La tâche peut être configurée systématiquement** en post-traitement de
toutes les règles de réception PeSIT, sans risque :

- Si l'émetteur a demandé un acquittement (via ``Reply To (ACK)`` sur son
  partenaire), la tâche envoie le F.MESSAGE automatiquement.
- Si l'émetteur n'a pas demandé d'acquittement (aucune information de retour
  disponible), la tâche est **silencieusement ignorée** — le transfert se
  termine normalement sans erreur.
- Si le protocole du transfert n'est pas PeSIT, la tâche est également
  ignorée.

Il n'est donc **pas nécessaire d'ajouter une condition** à la tâche pour
gérer ces cas : le comportement est automatique.

Paramètres
----------

* **partner** (*string*, optionnel) — Le nom du partenaire PeSIT distant vers
  lequel envoyer le message. Si omis, résolu automatiquement depuis la
  configuration ``Reply To (ACK)`` du partenaire émetteur. Supporte la
  substitution de variables.
* **account** (*string*, optionnel) — Le login du compte distant à utiliser
  pour l'authentification. Si omis, résolu depuis la configuration
  ``Reply To (ACK)`` ou le premier compte disponible sur le partenaire.
* **message** (*string*, optionnel) — Le contenu du F.MESSAGE. Supporte la
  substitution de variables (``#TRUEFILENAME#``, ``#TRANSFERID#``, etc.).
  Maximum 4096 caractères.
* **transferId** (*string*, optionnel) — L'identifiant de transfert PeSIT à
  référencer dans le F.MESSAGE. Si omis, utilise l'identifiant distant du
  transfert en cours. Supporte la substitution de variables.
* **passthrough** (*string*, optionnel) — Si ``"true"``, la tâche n'envoie
  aucun message et n'enregistre aucun suivi ACK. Utilisé sur les **nœuds
  relais** en Store & Forward pour que seul le destinataire final émette
  l'ACK (voir `Mode passthrough (relais S&F)`_).
* **customerID** (*string*, optionnel) — Identifiant applicatif transmis dans
  le F.MESSAGE. En Store & Forward, permet de propager l'identité de
  l'émetteur original à travers la chaîne de relais.
* **bankID** (*string*, optionnel) — Identifiant applicatif secondaire.
  Alternative au Customer ID pour identifier l'émetteur original dans la
  chaîne de relais.

Exemples
--------

**Acquittement automatique (zéro argument)** — le plus simple, recommandé
lorsque ``Reply To (ACK)`` est configuré sur le partenaire émetteur :

.. code-block:: yaml

   post:
     - type: SENDMESSAGE

**Acquittement avec message personnalisé** :

.. code-block:: yaml

   post:
     - type: SENDMESSAGE
       args:
         message: "Fichier #TRUEFILENAME# recu avec succes"

**Acquittement explicite (partenaire et compte spécifiés)** :

.. code-block:: yaml

   post:
     - type: SENDMESSAGE
       args:
         partner: "partenaire-emetteur"
         account: "mon-login"
         message: "ACK transfer #TRANSFERID#"

.. note::

   Si aucun partenaire n'est résolvable (pas de ``partner`` explicite, pas de
   ``Reply To (ACK)`` configuré sur le partenaire émetteur), la tâche est
   **silencieusement ignorée**. La tâche ``SENDMESSAGE`` peut donc être
   configurée systématiquement en post-traitement sans risque : elle ne
   s'exécute que quand un ACK a été demandé.

Mode passthrough (relais S&F)
-----------------------------

En Store & Forward (A → B → C), le nœud relais (B) ne doit **pas** envoyer
son propre ACK à l'émetteur (A). Seul le destinataire final (C) doit émettre
l'acquittement, qui sera automatiquement relayé par B vers A.

Pour cela, la tâche ``SENDMESSAGE`` du nœud relais doit être configurée avec
``passthrough: "true"`` :

.. code-block:: yaml

   # Sur le nœud relais (B) — règle de réception
   post:
     - type: TRANSFER
       args:
         to: "partenaire-c"
         as: "mon-login"
         using: "pesit-client"
         rule: "regle-reception"
         file: "#TRUEFULLPATH#"
     - type: SENDMESSAGE
       args:
         passthrough: "true"

**Comportement avec passthrough** :

- Aucun F.MESSAGE n'est envoyé par le relais
- Aucune entrée de suivi ACK n'est créée
- L'acquittement du destinataire final (C) est automatiquement relayé vers
  l'émetteur (A) via le mécanisme ``relayMessages`` du serveur PeSIT

**Sans passthrough (défaut)** — à utiliser sur le **destinataire final** :

.. code-block:: yaml

   # Sur le destinataire final (C) — règle de réception
   post:
     - type: SENDMESSAGE

Suivi ACK (ack_tracking)
------------------------

La Gateway enregistre automatiquement l'état du suivi ACK dans une table
dédiée ``ack_tracking``. Cet état est visible dans l'interface web sous forme
de badge coloré sur la page de monitoring des transferts :

.. list-table::
   :header-rows: 1

   * - Badge
     - État
     - Signification
   * - Rouge
     - En attente
     - L'émetteur a demandé un ACK mais ne l'a pas encore reçu
   * - Bleu
     - Émis
     - Le récepteur a envoyé l'ACK au partenaire
   * - Vert
     - Reçu
     - L'émetteur a reçu l'ACK (avec l'identité du destinataire final)

Le détail de l'acquittement (origine, date, message, identifiants) est
visible dans la modale « Plus d'info » du transfert.

Demander un acquittement
------------------------

Pour activer le suivi des acquittements sur un partenaire, renseigner le
champ **Acquittement (ACK)** (``replyTo`` en JSON/YAML) dans la configuration
protocolaire PeSIT du partenaire distant. La valeur est l'adresse de retour
au format ``serveur-local:compte``.

Ce champ a un double rôle :

- **Activer le suivi** : tous les transferts vers ce partenaire auront un
  badge rouge (ACK en attente) jusqu'à réception de l'acquittement (badge
  vert).
- **Fournir l'adresse de retour** (mode standard uniquement) : le récepteur
  utilise cette adresse pour savoir où envoyer l'ACK.

**Mode standard (entre Gateways Waarp)** : l'adresse de retour est transmise
dans la connexion PeSIT. La tâche ``SENDMESSAGE`` du récepteur résout
automatiquement le partenaire et le compte depuis cette adresse.

**Mode historique** : le partenaire distant connaît
déjà la route de retour par sa propre configuration. Le champ ``Acquittement``
sert uniquement à activer le suivi ACK côté Gateway. L'adresse renseignée
n'est pas utilisée par le partenaire distant (elle est transmise dans le
freetext PeSIT, que les produits historiques ignorent).

Résolution du partenaire
~~~~~~~~~~~~~~~~~~~~~~~~

Lorsque le paramètre ``partner`` de la tâche ``SENDMESSAGE`` n'est pas
renseigné :

1. Résolu depuis la configuration ``Acquittement (ACK)`` du partenaire
   émetteur (mode standard)
2. Résolu depuis le Customer ID / Bank ID du transfert (mode historique)

Lorsque le paramètre ``account`` n'est pas renseigné :

1. Compte extrait de la configuration ``Acquittement (ACK)``
2. Premier compte disponible sur le partenaire résolu

Intégration Store & Forward
----------------------------

Pour une chaîne A → B → C avec acquittement de bout en bout :

1. **Émetteur (A)** : renseigner le champ ``Acquittement (ACK)`` dans les
   paramètres PeSIT du partenaire distant (B), avec l'adresse de retour
   (serveur:compte)
2. **Relais (B)** :

   - Activer ``relayMessages: true`` dans la configuration du serveur PeSIT
   - Renseigner le champ ``Acquittement (ACK)`` sur le partenaire distant
     (C) pour le suivi ACK du maillon B→C
   - Configurer la règle de réception avec ``TRANSFER`` + ``SENDMESSAGE``
     en mode ``passthrough``

3. **Destinataire final (C)** : ajouter ``SENDMESSAGE`` en post-traitement
   de la règle de réception

L'acquittement du destinataire final (C) est automatiquement relayé à travers
la chaîne jusqu'à l'émetteur (A), avec l'identité de C préservée.

Compatibilité
~~~~~~~~~~~~~~

Le mécanisme d'acquittement est compatible avec toute implémentation PeSIT
conforme au standard :

- **Mode standard** : les deux partenaires sont des Gateways Waarp — utiliser
  ``SENDMESSAGE`` en post-traitement et ``Acquittement (ACK)`` sur le
  partenaire.
- **Mode historique** : le partenaire distant est un produit tiers — configurer
  l'envoi de F.MESSAGE côté partenaire (selon sa propre documentation) et
  activer le champ ``Acquittement (ACK)`` côté Gateway pour le suivi.

Voir la section :ref:`ref-proto-pesit` pour le détail de la configuration
des partenaires et serveurs PeSIT.
