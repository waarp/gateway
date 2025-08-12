.. _ref-cli-client-email:

##############################
Gestion des paramètres d'email
##############################

Identifiants SMTP
=================

La commande de gestion les identifiants SMTP est ``email credential``.

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

   credentials/add
   credentials/list
   credentials/get
   credentials/update
   credentials/delete

Templates d'email
=================

La commande de gestion des templates d'email est ``email template``.

.. tip::

   Le sujet, corps et pièces jointes de l'email peuvent contenir des
   :ref:`variables de substitution <reference-tasks-substitutions>` pour
   rendre leur contenu dynamique.

**Sommaire**

.. toctree::
   :maxdepth: 1

   templates/add
   templates/list
   templates/get
   templates/update
   templates/delete