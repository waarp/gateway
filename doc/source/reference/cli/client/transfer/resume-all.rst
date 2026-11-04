==============================
Reprendre plusieurs transferts
==============================

.. program:: waarp-gateway transfer resume-all

Reprend, là où ils se sont arrêtés, tous les transferts qui remplissent les
critères donnés en option, en une seule requête : les transferts que
``transfer list`` afficherait avec les mêmes filtres, moins ceux exclus par
``--except``. Sans aucune option, l'ensemble est celui de tous les transferts.

La commande affiche combien de transferts ont repris, combien ont été laissés
de côté et pourquoi (un transfert serveur reprend à l'initiative du partenaire,
l'enfant d'un transfert synchrone reprend par son parent, un transfert terminé
ne reprend pas).

**Commande**

.. code-block:: shell

   waarp-gateway transfer resume-all

**Options**

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
   égale à celle renseignée, au format ISO 8601 tel qu'il est décrit dans la
   :rfc:`3339`.

.. option:: -f <FOLLOW_ID>, --follow-id=<FOLLOW_ID>

   Restreint l'ensemble aux transferts ayant l'ID de flux renseigné.

.. option:: --except=<ID>

   Exclut de l'ensemble le transfert portant cet identifiant. L'option peut
   être renseignée plusieurs fois.

**Exemple**

.. code-block:: shell

   waarp-gateway transfer resume-all --rule commandes --status ERROR --except 12
