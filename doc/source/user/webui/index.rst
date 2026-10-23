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

   accueil
   serveurs-locaux
   clients-locaux
   partenaires-distants
   regles
   cles
   smtp
   modeles
   utilisateurs
   instances-cloud
   autorites
   snmp
   surcharges
   etat-des-services
   suivi-des-transferts


Connexion
=========

L'interface demande un identifiant et un mot de passe : ceux d'un
:ref:`utilisateur <user-webui-users>` de la Gateway, les mêmes que pour l'API
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
     - Produit, nom de l'instance, langue, lien vers la documentation (ouverte
       dans un nouvel onglet), menu du compte.
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
L'écran :doc:`serveurs-locaux`
requiert les droits sur les serveurs : lecture pour la liste et le détail,
écriture pour créer, modifier, démarrer et arrêter, suppression pour
supprimer. L'écran :doc:`partenaires-distants` requiert de même les droits
sur les partenaires, l'écran :doc:`utilisateurs` les droits sur les
utilisateurs, et les écrans :doc:`instances-cloud`, :doc:`autorites`,
:doc:`snmp`, :doc:`surcharges` et :doc:`cles` les droits sur l'administration ;
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
