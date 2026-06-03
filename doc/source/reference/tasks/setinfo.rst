.. _ref-task-setinfo:

SETINFO
=======

Le traitement ``SETINFO`` permet de positionner, modifier ou supprimer une clé
dans les :term:`infos de transfert<infos de transfert>` du transfert en cours.

Cette tâche est utile en pré-traitement pour injecter des métadonnées avant un
rebond via la tâche ``TRANSFER`` (avec ``copyInfo: true``), ou pour conditionner
l'exécution de tâches suivantes via le mécanisme de :ref:`conditions <ref-task-sendmessage>`.

Paramètres
----------

* **key** (*string*, **obligatoire**) — La clé TransferInfo à positionner.
  Exemples : ``__fileEncoding__``, ``__replyPartner__``, ou toute clé
  personnalisée.
* **value** (*string*, optionnel) — La valeur à affecter à la clé. Supporte
  la substitution de variables (``#TRUEFILENAME#``, ``#TRANSFERID#``, etc.).
  Si la valeur est **vide**, la clé est **supprimée** du TransferInfo.

Exemples
--------

**Injecter l'encodage avant un rebond** :

.. code-block:: yaml

   pre:
     - type: SETINFO
       args:
         key: "__fileEncoding__"
         value: "EBCDIC"
     - type: TRANSFER
       args:
         to: "destinataire"
         copyInfo: true

**Positionner l'adresse de retour manuellement** (quand PI 99 REPLY= n'est pas
disponible) :

.. code-block:: yaml

   pre:
     - type: SETINFO
       args:
         key: "__replyPartner__"
         value: "partenaire-emetteur"
     - type: SETINFO
       args:
         key: "__replyAccount__"
         value: "mon-login"

**Supprimer une clé** (valeur vide) :

.. code-block:: yaml

   pre:
     - type: SETINFO
       args:
         key: "__tempKey__"
         value: ""

**Injecter un identifiant métier dynamique** :

.. code-block:: yaml

   pre:
     - type: SETINFO
       args:
         key: "batchId"
         value: "BATCH-#DATE#-#HOUR#"
