

###################################
Gestion de la configuration d'email
###################################

Cette section documente la configuration des paramètres d'email, utilisés dans la
:ref:`tâche EMAIL<ref-task-email>`.

.. _ref-rest-smtp:

Identifiants SMTP
=================

Le point d'accès pour gérer les identifiants SMTP est ``/api/email/credentials``.

.. note::

   L'authentification SMTP se fait via l'extension "AUTH" du protocole, tel qu'elle
   est décrite dans la :rfc:`4954`. Les mécanismes d'authentification supportés
   sont : "PLAIN", "LOGIN" et "CRAM-MD5".

   Si le serveur cible supporte l'extension "STARTTLS", la connexion sera alors
   *upgradée* en TLS, afin d'éviter de transmettre ces identifiants en clair.

   Si aucun login n'est renseigné dans un identifiant SMTP, alors la connexion
   ne sera pas authentifiée (à supposer que le serveur le permette).

**Sommaire**

.. toctree::
   :maxdepth: 1

   credentials/create
   credentials/list
   credentials/consult
   credentials/update
   credentials/delete

.. _ref-rest-emails:

Templates d'email
=================

Le point d'accès pour gérer les templates d'email est ``/api/email/templates``.

.. tip::

   Le sujet, corps et pièces jointes de l'email peuvent contenir des
   :ref:`variables de substitution <reference-tasks-substitutions>` pour
   rendre leur contenu dynamique.

**Sommaire**

.. toctree::
   :maxdepth: 1

   templates/create
   templates/list
   templates/consult
   templates/update
   templates/delete


