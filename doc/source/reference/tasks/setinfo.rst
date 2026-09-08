.. _ref-task-setinfo:

SETINFO
=======

.. versionadded:: 0.16.0

Le traitement ``SETINFO`` permet de positionner, modifier ou supprimer une clé
dans les :term:`infos de transfert<infos de transfert>` du transfert en cours.

Cette tâche est utile en pré-traitement pour injecter des métadonnées avant un
rebond via la tâche ``TRANSFER`` (avec ``copyInfo: true``).

Paramètres
----------

* **key** (*string*) — [DÉPRÉCIÉ] La clé TransferInfo à positionner.
* **value** (*string*) — [DÉPRÉCIÉ] La valeur à affecter à la clé. Supporte
  la substitution de variables (``#TRUEFILENAME#``, ``#TRANSFERID#``, etc.).
  Si cette valeur est omise ou nulle, la clé est supprimée du TransferInfo.

.. versionchanged:: 0.17.0

   Les informations de transfert peuvent être données tel quel sous forme d'un
   objet JSON ou YAML. Les clés "key" et "value" restent réservées pour maintenir
   la rétro-compatibilité, mais elles ne sont plus obligatoires. Si une clé est
   spécifiée avec une valeur vide (``""``) ou nulle (``"null"``), alors la clé
   sera supprimée des informations de transfert.

Exemples
--------

**Injecter un identifiant métier dynamique** :

.. code-block:: yaml

   pre:
     - type: SETINFO
       args:
         batchId: "BATCH-#DATE#-#HOUR#"

**Supprimer une clé** (valeur nulle) :

.. code-block:: yaml

   pre:
     - type: SETINFO
       args:
         "__tempKey__": "null"



