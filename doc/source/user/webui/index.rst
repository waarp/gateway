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

   connexion
   accueil
   serveurs-locaux
   clients-locaux
   partenaires-distants
   regles
   surveillances
   cles
   smtp
   modeles
   utilisateurs
   instances-cloud
   autorites
   snmp
   surcharges
   import-export
   etat-des-services
   suivi-des-transferts


Connexion et langue
===================

L'interface demande un identifiant et un mot de passe : ceux d'un
:ref:`utilisateur <user-webui-users>` de la Gateway, les mêmes que pour l'API
REST et la ligne de commande. La page :doc:`connexion` décrit la connexion, le
refus et le verrouillage, la session et la déconnexion.

L'interface existe en français et en anglais. Le sélecteur de langue se trouve
dans l'en-tête, et en haut à droite de la page de connexion ; le choix est
conservé dans un cookie. Sans choix, la langue est celle du navigateur.


Navigation
==========

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Zone
     - Contenu
   * - En-tête
     - Produit, nom de l'instance, langue, lien vers la documentation (ouverte
       dans un nouvel onglet), menu du compte.
   * - Barre latérale
     - Les écrans, groupés par section : Connexions, Traitements,
       Supervision, Administration. Un groupe se replie d'un clic sur son
       titre. Le bouton de l'en-tête replie la barre en une colonne
       d'icônes ; le libellé d'un écran s'affiche alors au survol ou au
       focus. Les états de repli sont mémorisés.
   * - Fil d'Ariane
     - Le chemin de l'écran courant, depuis l'accueil.
   * - Contenu
     - L'écran lui-même : en-tête de page (section, titre, description,
       action), puis tableaux et cartes.

Le lien « Aller au contenu », premier élément atteint par la touche
:kbd:`Tab`, mène directement au contenu de l'écran.


.. _user-webui-search:

Recherche dans les listes
=========================

Le champ de recherche d'une liste cherche le texte saisi n'importe où dans le
nom, l'identifiant ou l'adresse qu'il indique, majuscules, minuscules et
accents confondus : « prod » trouve « Banque-PROD » comme « production ».
Quatre caractères ont un sens particulier.

.. list-table::
   :header-rows: 1
   :widths: 15 85

   * - Caractère
     - Effet
   * - ``*``
     - Remplace un texte quelconque, même vide : ``sftp*prod`` trouve
       « sftp-banque-prod ».
   * - ``?``
     - Remplace un caractère, exactement un : ``v?rements`` trouve
       « virements ».
   * - ``^``
     - En tête, cale la recherche au début : ``^srv`` trouve les noms qui
       commencent par « srv ».
   * - ``$``
     - En queue, cale la recherche à la fin : ``prod$`` trouve les noms qui
       finissent par « prod ».

Dans le suivi des transferts, la recherche porte sur le nom et le chemin du
fichier, tels que la liste les affiche, majuscules et minuscules confondues
mais accents compris, et sur l'identifiant de suivi, exact.


Pages de création et de modification
====================================

Une page de création ou de modification dont le formulaire a changé sans être
validé demande confirmation avant d'être quittée. Un lien de l'interface
(barre latérale, fil d'Ariane, bouton de retour) ouvre la fenêtre *Quitter
sans enregistrer ?*, qui propose de rester sur la page ou de la quitter sans
enregistrer. La fermeture de l'onglet, le retour du navigateur et le
rechargement ouvrent la fenêtre du navigateur, la seule qu'il permette dans ces
cas. Rien n'est demandé si rien n'a été saisi, à la validation, ni sur
*Annuler*. Un formulaire refusé, un champ en erreur par exemple, n'est pas
enregistré : le quitter demande aussi confirmation.


Droits
======

La barre latérale ne montre que les écrans que l'utilisateur a le droit de
consulter, selon ses :ref:`droits <user-webui-users>` : la consultation d'un
écran requiert le droit de lecture de sa catégorie, un ajout ou une
modification le droit d'écriture, une suppression le droit de suppression. Le
serveur contrôle ces droits sur chaque page et chaque action, indépendamment
de ce que l'interface affiche ; une requête non autorisée reçoit le code 403.

L':doc:`accueil` et l'écran :doc:`etat-des-services` ne requièrent aucun
droit particulier : tout utilisateur authentifié peut les consulter ;
l'accueil ne propose que les écrans que l'utilisateur a le droit de lire.
Les écrans :doc:`serveurs-locaux` et :doc:`surveillances`
requièrent les droits sur les serveurs : lecture pour la liste et le détail,
écriture pour créer, modifier, démarrer et arrêter, suppression pour
supprimer. L'écran :doc:`partenaires-distants` requiert de même les droits
sur les partenaires, l'écran :doc:`utilisateurs` les droits sur les
utilisateurs, et les écrans :doc:`instances-cloud`, :doc:`autorites`,
:doc:`snmp`, :doc:`surcharges`, :doc:`import-export` et :doc:`cles` les droits
sur l'administration ;
les écrans :doc:`smtp` et :doc:`modeles` requièrent les droits sur les règles,
comme l'API REST. L'écran :doc:`suivi-des-transferts` requiert les droits sur
les transferts : lecture pour la liste et la fiche, écriture pour toute
action.


Ancienne et nouvelle interface
==============================

L'interface est refondue écran par écran. Un écran refondu remplace l'ancien
dans la navigation et sur la page d'accueil dès sa livraison ; l'ancien écran
reste accessible à son adresse jusqu'à la version suivante, puis est retiré.
Les écrans non encore refondus conservent leur présentation historique et
restent accessibles depuis la barre latérale.
