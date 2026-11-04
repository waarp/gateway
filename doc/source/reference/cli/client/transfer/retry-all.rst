===============================
Retenter plusieurs transferts
===============================

.. program:: waarp-gateway transfer retry-all

Rejoue en intégralité tous les transferts qui remplissent les critères donnés
en option, en une seule requête : les transferts que ``transfer list``
afficherait avec les mêmes filtres, moins ceux exclus par ``--except``. Chaque
transfert rejoué donne un nouveau transfert, planifié à la date donnée ou
immédiatement. Sans aucune option de filtre, l'ensemble est celui de tous les
transferts.

La commande affiche combien de transferts ont été rejoués, combien ont été
laissés de côté et pourquoi (un transfert serveur est rejoué par le partenaire,
un transfert en cours ou en attente ne se rejoue pas, un transfert dont la
règle, le partenaire, le compte ou le client n'existe plus), puis les
identifiants des nouveaux transferts.

**Commande**

.. code-block:: shell

   waarp-gateway transfer retry-all

**Options**

.. option:: -d <DATE>, --date=<DATE>

   La date à laquelle les nouveaux transferts démarreront. Par défaut, ils
   démarrent immédiatement. La date doit être renseignée en suivant le format
   standard ISO 8601 tel qu'il est décrit dans la :rfc:`3339`.

.. option:: -r <RULE>, --rule=<RULE>

   Restreint l'ensemble aux transferts utilisant la règle renseignée. L'option
   peut être renseignée plusieurs fois pour plusieurs règles.

.. option:: --status=<STATUS>

   Restreint l'ensemble aux transferts ayant le statut renseigné. L'option peut
   être renseignée plusieurs fois pour plusieurs statuts. Les statuts valides
   sont : ``PLANNED``, ``RUNNING``, ``PAUSED``, ``INTERRUPTED``, ``ERROR``,
   ``CANCELLED`` et ``DONE``.

.. option:: --start=<DATE>

   Restreint l'ensemble aux transferts dont la date de début est postérieure ou
   égale à celle renseignée (:rfc:`3339`).

.. option:: -f <FOLLOW_ID>, --follow-id=<FOLLOW_ID>

   Restreint l'ensemble aux transferts ayant l'ID de flux renseigné.

.. option:: --except=<ID>

   Exclut de l'ensemble le transfert portant cet identifiant. L'option peut
   être renseignée plusieurs fois.

**Exemple**

.. code-block:: shell

   waarp-gateway transfer retry-all --status CANCELLED --start "2026-11-04T00:00:00+01:00" -d "2026-11-05T06:00:00+01:00"
