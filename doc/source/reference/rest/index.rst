.. _reference-rest-api:

########
API REST
########

L'interface REST de Gateway peut être accédée à partir de la racine ``/api``
en interrogeant l'adresse d'administration donnée dans la configuration.

.. note:: 
   Toutes les dates doivent être renseignées en format ISO 8601 tel
   qu'il est spécifié dans la :rfc:`3339`.

.. note::
   Il existe 2 type de mise-à-jour pour un objet REST : la mise-à-jour partielle
   et la mise-à-jour complète (aussi appelés respectivement modification et
   remplacement).
   Une mise-à-jour partielle se fait via une requête HTTP PATCH. Une mise-à-jour
   complète se fait, elle, via une requête HTTP PUT.
   Lors d'une mise à jours partielle, les champs omit (ou nuls) restent inchangés,
   et conservent donc leur ancienne valeur. Lors d'une mise à jour complète,
   les champs omits sont réinitialisés à leurs valeurs par défaut.

.. important::
   Pour une meilleure lisibilité, les URLs donnés dans les exemples de requêtes
   et de réponses de cette documentation ne sont pas encodés en `encodage-pourcent
   <https://fr.wikipedia.org/wiki/Encodage-pourcent>`_. Cependant, dans la pratique,
   il est impératif que les éventuels caractères réservés présents dans vos URLs
   soient échappés via ce mécanisme, sans quoi ces URLs ne seront pas valides.
   Veillez donc bien à passer vos URLs dans un encodeur d'URL avant de les
   envoyer dans des requêtes.

.. toctree::
   :maxdepth: 1

   authentication
   status
   about
   users/index
   transfers/index
   history/index
   servers/index
   clients/index
   partners/index
   rules/index
   clouds/index
   override/index
   authorities/index
   snmp/index
   keys/index

