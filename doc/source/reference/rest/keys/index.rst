.. _rest_keys:

Gestion des clés cryptographiques
#################################

Le point d'accès pour gérer les :term:`clés cryptographiques<clé cryptographique>`
est ``/api/keys``. Les clés cryptographiques servent a effectuer des tâches
cryptographique.

Les types de clé acceptés sont :

- ``AES`` pour les clés de (dé)chiffrement AES
- ``HMAC`` pour les clés de signature HMAC
- ``PGP-PUBLIC`` pour les clés PGP publiques
- ``PGP-PRIVATE`` pour les clés PGP privées

Pour les clés AES, la longueur de la clé déterminera la version de AES utilisée,
à savoir, 16 octets pour AES-128, 24 octets pour AES-192, et 32 octets pour
AES-256.

.. toctree::
   :maxdepth: 1

   create
   list
   consult
   update
   delete