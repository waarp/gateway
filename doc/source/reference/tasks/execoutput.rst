EXECOUTPUT
==========

Le traitement 'EXECOUTPUT' exécute un programme externe. Si une erreur survient
durant exécution, les lignes écrites dans la sortie standard seront interprétées
comme message d'erreur. Il est également possible pour le programme de déplacer
le fichier de transfert durant l'exécution. Dans ce cas, la dernière ligne
écrite sur la sortie standard par le programme doit commencer par
"``NEWFILENAME:``", suivi du nouveau chemin du fichier.

* **path** (*string*) - Le chemin du programme à exécuter.
* **args** (*string*) - Les arguments du programme.
* **delay** (*number*) - La durée d'exécution maximum du programme (en ms). Si
  le programme n'a pas terminé après que cette durée soit écoulée, le programme
  sera interrompu et le traitement sera considéré comme ayant échoué.

.. note::
   La valeur de sortie du programme détermine si l'exécution a réussie ou échoué.
   - une valeur de 0 signifie un succès
   - une valeur de 1 signifie un succès avec message d'avertissement
   - toute autre valeur est considérée comme un échec