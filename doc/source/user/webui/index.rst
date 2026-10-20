.. _user-guide-webui:

###############
Interface web
###############

Waarp Gateway embarque une interface web d'administration. Elle est servie par
le service d'administration, sur la même adresse et le même port que l'API
REST (section ``[admin]`` du :ref:`fichier de configuration
<configuration-file>`), à l'adresse ``http://<hôte>:<port>/webui/``.

Ce guide décrit l'interface écran par écran. Il naît avec la refonte de
l'interface : chaque écran refondu y ajoute sa page, les écrans historiques
restent décrits par l'interface elle-même.

.. toctree::
   :maxdepth: 1

   etat-des-services


Connexion
=========

L'interface demande un identifiant et un mot de passe : ceux d'un
:ref:`utilisateur <user-management>` de la Gateway, les mêmes que pour l'API
REST et la ligne de commande. Après 5 échecs pour un même compte depuis une
même adresse, les tentatives sont refusées pendant 1 minute (section
``[security]`` de la configuration).

La session reste ouverte tant que l'utilisateur agit ; elle expire après 20
minutes sans action et, dans tous les cas, après 24 heures. Le menu du compte,
en haut à droite, permet de se déconnecter.


Langue
======

L'interface existe en français et en anglais. Le sélecteur de langue se trouve
dans l'en-tête ; le choix est conservé dans un cookie. Sans choix, la langue
est celle du navigateur.


Navigation
==========

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Zone
     - Contenu
   * - En-tête
     - Produit, nom de l'instance, langue, lien vers la documentation, menu du
       compte.
   * - Barre latérale
     - Les écrans, groupés par section : Connexions, Traitements,
       Supervision, Administration. Un groupe se replie d'un clic sur son
       titre ; l'état de repli est mémorisé.
   * - Fil d'Ariane
     - Le chemin de l'écran courant, depuis l'accueil.
   * - Contenu
     - L'écran lui-même : en-tête de page (section, titre, description,
       action), puis tableaux et cartes.

Le lien « Aller au contenu », premier élément atteint par la touche
:kbd:`Tab`, mène directement au contenu de l'écran.


Droits
======

La barre latérale ne montre que les écrans que l'utilisateur a le droit de
consulter, selon ses :ref:`droits <user-management>` : la consultation d'un
écran requiert le droit de lecture de sa catégorie, un ajout ou une
modification le droit d'écriture, une suppression le droit de suppression. Le
serveur contrôle ces droits sur chaque page et chaque action, indépendamment
de ce que l'interface affiche ; une requête non autorisée reçoit le code 403.

L'écran :doc:`etat-des-services` ne requiert aucun droit particulier : tout
utilisateur authentifié peut le consulter.


Ancienne et nouvelle interface
==============================

L'interface est refondue écran par écran. Un écran refondu remplace l'ancien
dans la navigation et sur la page d'accueil dès sa livraison ; l'ancien écran
reste accessible à son adresse jusqu'à la version suivante, puis est retiré.
Les écrans non encore refondus conservent leur présentation historique et
restent accessibles depuis la barre latérale.
