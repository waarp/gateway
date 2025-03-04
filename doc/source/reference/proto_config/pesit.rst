.. _proto-config-pesit:

Configuration PeSIT & PeSIT-TLS
###############################

Configuration client
====================

Le configuration protocolaire d'un client affectera tous les transferts effectués
avec ce client. Il est possible d'écraser cette configuration au cas par cas via
la configuration des partenaires (voir ci-dessous). La structure de l'objet JSON
de configuration du protocole pour un client PeSIT est la suivante :

* **disableRestart** (*boolean*) - Désactive le "restart" pour tous les transferts
  effectués avec ce client. Par défaut, le "restart" est activé. Cette valeur
  peut être écrasée au cas par cas dans la configuration des partenaires
  (voir ci-dessous).
* **disableCheckpoints** (*boolean*) - Désactive les checkpoints pour tous les
  transferts effectués avec ce client. Par défaut, les checkpoints sont activés.
  Cette valeur peut être écrasée au cas par cas dans la configuration des
  partenaires (voir ci-dessous).
* **checkpointSize** (*integer*) - Spécifie la taille (en octets) des bloc de
  données entre chaque checkpoint lors d'un transfert. N'a aucun effet si les
  checkpoints sont désactivés. Par défaut, les blocs entre checkpoints font
  65535 octets.
* **checkpointWindow** (*integer*) - Spécifie le nombre de checkpoints pouvant
  rester sans réponse avant que le transfert soit stoppé. N'a aucun effet si
  les checkpoints sont désactivés. Par défaut, le transfert sera stoppé si 2
  checkpoints restent sans réponse du partenaire.

**Exemple**

.. code-block:: json

   {
     "disableRestart": false,
     "disableCheckpoints": false,
     "checkpointSize": 65535,
     "checkpointWindow": 2
   }

Configuration partenaire
========================

La configuration protocolaire des partenaires PeSIT est identique à celle du
client. Cependant, si une option de la configuration du partenaire contredit la
configuration du client, la configuration du partenaire (plus spécifique)
est prioritaire. Si une option de la configuration du partenaire n'est pas
renseignée, la configuration du client est utilisée. La structure de l'objet
JSON de configuration du protocole pour un partenaire PeSIT est donc la suivante :

* **login** (*string*) - Le login du partenaire (optionnel). Le mot de passe du
  partenaire doit être renseigné via un :ref:`identifiant <reference-auth-methods>`
  de type *"password"* rattaché au partenaire.
* **disableRestart** (*boolean*) - Désactive le "restart" pour tous les transferts
  effectués avec ce partenaire. Par défaut, la valeur donnée dans la configuration
  du client est utilisée.
* **disableCheckpoints** (*boolean*) - Désactive les checkpoints pour tous les
  transferts effectués avec ce partenaire. Par défaut, la valeur donnée dans la
  configuration du client est utilisée.
* **checkpointSize** (*integer*) - Spécifie la taille (en octets) des bloc de
  données entre chaque checkpoint lors d'un transfert. N'a aucun effet si les
  checkpoints sont désactivés. Par défaut, la valeur donnée dans la configuration
  du client est utilisée.
* **checkpointWindow** (*integer*) - Spécifie le nombre de checkpoints pouvant
  rester sans réponse avant que le transfert soit stoppé. N'a aucun effet si
  les checkpoints sont désactivés. Par défaut, la valeur donnée dans la
  configuration du client est utilisée.
* **useNSDU** (*boolean*) - Spécifie si les méta-paquets NSDU du protocole PeSIT
  doivent être utilisés lors des transferts avec ce partenaire. Par défaut, les
  paquets NSDU ne sont pas utilisés.
* **compatibilityMode** (*string*) - Spécifie le mode de compatibilité à utiliser
  lors des communications avec le partenaire, dans le cas où celui-ci ne respecterait
  pas la spécification du protocole PeSIT. Les valeurs autorisés sont : ``none``
  et ``axway``. Par défaut, aucun mode de compatibilité n'est utilisé (``none``).
* **maxMessageSize** (*integer*) - Spécifie la taille maximale (en octets) autorisée
  pour les paquets PeSIT envoyés à (et reçus depuis) ce partenaire. Le partenaire
  pourra unilatéralement décider d'utiliser une taille plus petite que celle-ci,
  mais jamais plus grande. La valeur par défaut est de 65535 octets.
* **disablePreConnection** (*boolean*) - Permet de désactiver le processus de
  pré-connexion (et la pré-authentification qui va avec) pour ce partenaire. Par
  défaut, un échange de pré-connexion est attendu à chaque nouvelle connexion.
* **minTLSVersion** (*string*) - [PeSIT-TLS uniquement] Spécifie la version
  minimale de TLS autorisée pour ce partenaire. Par défaut, la valeur "v1.2"
  (pour TLS 1.2) est utilisée.

**Exemple**

.. code-block:: json

   {
     "disableRestart": false,
     "disableCheckpoints": false,
     "checkpointSize": 65535,
     "checkpointWindow": 2,
     "useNSDU": true,
     "compatibilityMode": "axway",
     "maxMessageSize": "65535",
     "minTLSVersion": "v1.2"
   }

Configuration serveur
=====================

La structure de l'objet JSON de configuration du protocole pour un serveur PeSIT
est la suivante :

* **disableRestart** (*boolean*) - Désactive le "restart" pour tous les transferts
  effectués avec ce serveur. Par défaut, le "restart" est activé.
* **disableCheckpoints** (*boolean*) - Désactive les checkpoints pour tous les
  transferts effectués avec ce serveur. Par défaut, les checkpoints sont activés.
* **checkpointSize** (*integer*) - Spécifie la taille maximale (en octets) des 
  blocs de données entre chaque checkpoint lors d'un transfert. Si un client se
  connectant au serveur demande une taille plus grande, celle-ci sera rabaissée
  à ce maximum. N'a aucun effet si les checkpoints sont désactivés. Par défaut,
  les blocs entre checkpoints font 65535 octets.
* **checkpointWindow** (*integer*) - Spécifie le nombre maximum de checkpoints 
  pouvant rester sans réponse avant que le transfert soit stoppé. Si un client se
  connectant au serveur demande un interval plus grand, celui-ci sera rabaissé
  à ce maximum. N'a aucun effet si les checkpoints sont désactivés. Par défaut,
  le transfert sera stoppé si 2 checkpoints restent sans réponse du récepteur.
* **maxMessageSize** (*integer*) - Spécifie la taille maximale (en octets) autorisée
  pour les paquets PeSIT envoyés à (et reçus depuis) ce serveur. Si un client se
  connectant au serveur demande une taille plus grande, celle-ci sera rabaissée
  à ce maximum. La valeur par défaut est de 65535 octets.
* **disablePreConnection** (*boolean*) - Permet de désactiver le processus de
  pré-connexion (et la pré-authentification qui va avec) si le partenaire client
  ne le supporte pas. Par défaut, un échange de pré-connexion aura lieu à chaque
  nouvelle connexion.
* **minTLSVersion** (*string*) - [PeSIT-TLS uniquement] Spécifie la version
  minimale de TLS autorisée par ce serveur. Par défaut, la valeur "v1.2"
  (pour TLS 1.2) est utilisée.

**Exemple**

.. code-block:: json

   {
     "disableRestart": false,
     "disableCheckpoints": false,
     "checkpointSize": 65535,
     "checkpointWindow": 1,
     "maxMessageSize**: 65535,
     "disablePreConnection": false,
     "minTLSVersion": "v1.2"
   }
